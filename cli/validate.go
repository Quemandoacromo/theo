package cli

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type ValidationResult string

const (
	ValResMismatch        = "mismatch"
	ValResError           = "error"
	ValResMissingChecksum = "missing checksum"
	ValResFileNotFound    = "file not found"
	ValResValid           = "valid"
)

var allValidationResults = []ValidationResult{
	ValResMismatch,
	ValResError,
	ValResMissingChecksum,
	ValResFileNotFound,
	ValResValid,
}

var valResMessageTemplates = map[ValidationResult]string{
	ValResMismatch:        "%s files did not match expected checksum",
	ValResError:           "%s files encountered errors during validation",
	ValResMissingChecksum: "%s files are missing checksums",
	ValResFileNotFound:    "%s files were not found",
	ValResValid:           "%s files are matching checksums",
}

func ValidateHandler(u *url.URL) error {
	ids := Ids(u)
	operatingSystems, langCodes, downloadTypes := OsLangCodeDownloadType(u)

	q := u.Query()

	var manualUrlFilter []string
	if q.Has("manual-url-filter") {
		manualUrlFilter = strings.Split(q.Get("manual-url-filter"), ",")
	}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	return Validate(operatingSystems, langCodes, downloadTypes, manualUrlFilter, rdx, ids...)
}

func Validate(operatingSystems []vangogh_integration.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_integration.DownloadType,
	manualUrlFilter []string,
	rdx redux.Writeable,
	ids ...string) error {

	va := nod.NewProgress("validating downloads...")
	defer va.Done()

	for _, id := range ids {

		productDetails, err := getProductDetails(id, rdx, false)
		if err != nil {
			return err
		}

		if mismatchedManualUrls, err := validateLinks(id, operatingSystems, langCodes, downloadTypes, manualUrlFilter, productDetails); err != nil {
			return err
		} else if len(mismatchedManualUrls) > 0 {

			// redownload and revalidate any manual-urls that resulted in mismatched checksums

			if err = Download(operatingSystems, langCodes, downloadTypes, mismatchedManualUrls, rdx, true, id); err != nil {
				return err
			}

			if _, err = validateLinks(id, operatingSystems, langCodes, downloadTypes, manualUrlFilter, productDetails); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateLinks(id string,
	operatingSystems []vangogh_integration.OperatingSystem,
	langCodes []string,
	downloadTypes []vangogh_integration.DownloadType,
	manualUrlFilter []string,
	productDetails *vangogh_integration.ProductDetails) ([]string, error) {

	vla := nod.NewProgress("validating %s...", productDetails.Title)
	defer vla.Done()

	downloadsDir, err := pathways.GetAbsDir(data.Downloads)
	if err != nil {
		return nil, err
	}

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(operatingSystems...).
		FilterLanguageCodes(langCodes...).
		FilterDownloadTypes(downloadTypes...)

	if len(dls) == 0 {
		return nil, errors.New("no links are matching operating params")
	}

	vla.TotalInt(len(dls))

	results := make([]ValidationResult, 0, len(dls))

	var mismatchedManualUrls []string

	for _, dl := range dls {
		if len(manualUrlFilter) > 0 && !slices.Contains(manualUrlFilter, dl.ManualUrl) {
			continue
		}

		vr, err := validateLink(id, &dl, downloadsDir)
		if err != nil {
			vla.Error(err)
		}

		if vr == ValResMismatch {
			mismatchedManualUrls = append(mismatchedManualUrls, dl.ManualUrl)
		}

		results = append(results, vr)
	}

	vla.EndWithResult(summarizeValidationResults(results))

	return mismatchedManualUrls, nil
}

func validateLink(id string, link *vangogh_integration.ProductDownloadLink, downloadsDir string) (ValidationResult, error) {

	dla := nod.NewProgress(" - %s...", link.LocalFilename)
	defer dla.Done()

	absDownloadPath := filepath.Join(downloadsDir, id, link.LocalFilename)

	var stat os.FileInfo
	var err error

	if stat, err = os.Stat(absDownloadPath); os.IsNotExist(err) {
		dla.EndWithResult(ValResFileNotFound)
		return ValResFileNotFound, nil
	}

	if link.Md5 == "" {
		dla.EndWithResult(ValResMissingChecksum)
		return ValResMissingChecksum, nil
	}

	dla.Total(uint64(stat.Size()))

	localFile, err := os.Open(absDownloadPath)
	if err != nil {
		return ValResError, err
	}

	h := md5.New()
	if err = dolo.CopyWithProgress(h, localFile, dla); err != nil {
		return ValResError, err
	}

	computedMd5 := fmt.Sprintf("%x", h.Sum(nil))
	if link.Md5 == computedMd5 {
		dla.EndWithResult(ValResValid)
		return ValResValid, nil
	} else {
		dla.EndWithResult(ValResMismatch)
		return ValResMismatch, nil
	}
}

func summarizeValidationResults(results []ValidationResult) string {

	desc := make([]string, 0)

	for _, vr := range allValidationResults {
		if slices.Contains(results, vr) {
			someAll := "some"
			if isSameResult(vr, results) {
				someAll = "all"
			}
			desc = append(desc, fmt.Sprintf(valResMessageTemplates[vr], someAll))
		}
	}

	return strings.Join(desc, "; ")
}

func isSameResult(exp ValidationResult, results []ValidationResult) bool {
	for _, vr := range results {
		if vr != exp {
			return false
		}
	}
	return true
}
