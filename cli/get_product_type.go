package cli

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/camino"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func productTypeRelDir(pt vangogh_integration.ProductType) camino.RelDir {
	switch pt {
	case vangogh_integration.GogDetails:
		return data.GogDetails
	case vangogh_integration.GogChecksums:
		return data.GogChecksums
	case vangogh_integration.GogFilenames:
		return data.GogFilenames
	case vangogh_integration.GogImages:
		return data.GogImages
	default:
		panic("no rel dir set for product type: " + pt.String())
	}
}

func productTypeRequest(id string, pt vangogh_integration.ProductType, rdx redux.Readable) (*http.Request, error) {
	switch pt {
	case vangogh_integration.GogChecksums:
		apiGogChecksumsPath := path.Join(data.ApiGogChecksumsPath, id)
		return data.VangoghApiRequest(http.MethodGet, apiGogChecksumsPath, nil, rdx)
	case vangogh_integration.GogFilenames:
		apiGogFilenamesPath := path.Join(data.ApiGogFilenamesPath, id)
		return data.VangoghApiRequest(http.MethodGet, apiGogFilenamesPath, nil, rdx)
	case vangogh_integration.GogImages:
		apiGogImagesPath := path.Join(data.ApiGogImagesPath)
		return data.VangoghApiRequest(http.MethodGet, apiGogImagesPath, nil, rdx)
	case vangogh_integration.GogDetails:
		apiMetadataGogDetailsPath := path.Join(data.ApiMetadataPath, pt.String(), id)
		return data.VangoghApiRequest(http.MethodGet, apiMetadataGogDetailsPath, nil, rdx)
	default:
		return nil, errors.New("http request not supported for " + pt.String())
	}
}

func getProductType(id string, pt vangogh_integration.ProductType, rdx redux.Writeable, force bool) (io.ReadCloser, error) {

	gpta := nod.Begin(" getting %s %s...", pt, id)
	defer gpta.Done()

	ptDir := camino.GetRel(productTypeRelDir(pt), data.Metadata)

	kvPt, err := kevlar.New(ptDir, pt.Ext())
	if err != nil {
		return nil, err
	}

	if kvPt.Has(id) && !force {

		var rcLocal io.ReadCloser
		rcLocal, err = kvPt.Get(id)
		if err != nil {
			return nil, err
		}

		gpta.EndWithResult("read local")
		return rcLocal, nil
	}

	if err = vangoghValidateSessionToken(rdx); err != nil {
		return nil, err
	}

	if err = fetchRemoteProductType(id, pt, rdx, kvPt); err != nil {
		return nil, err
	}

	if err = reduceProductType(id, pt, rdx, kvPt); err != nil {
		return nil, err
	}

	gpta.EndWithResult("fetched remote")

	return kvPt.Get(id)
}

func fetchRemoteProductType(id string, pt vangogh_integration.ProductType, rdx redux.Readable, kvPt kevlar.KeyValues) error {

	frpta := nod.Begin(" fetching remote %s %s...", pt, id)
	defer frpta.Done()

	req, err := productTypeRequest(id, pt, rdx)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("error fetching %s: %s", pt, resp.Status))
	}

	return kvPt.Set(id, resp.Body)
}

func reduceProductType(id string, pt vangogh_integration.ProductType, rdx redux.Writeable, kvPt kevlar.KeyValues) error {

	rpta := nod.Begin(" reducing %s...", pt)
	defer rpta.Done()

	switch pt {
	case vangogh_integration.GogDetails:
		return reduceGogDetails(id, kvPt, rdx)
	case vangogh_integration.GogChecksums:
		fallthrough
	case vangogh_integration.GogFilenames:
		fallthrough
	case vangogh_integration.GogImages:
		return nil
	default:
		return errors.New("reduction not supported for " + pt.String())
	}
}

func getGogChecksums(id string, rdx redux.Writeable, force bool) (map[string]string, error) {
	rcGogChecksums, err := getProductType(id, vangogh_integration.GogChecksums, rdx, force)
	if err != nil {
		return nil, err
	}
	defer rcGogChecksums.Close()

	var gogChecksums map[string]string
	if err = json.UnmarshalRead(rcGogChecksums, &gogChecksums); err != nil {
		return nil, err
	}

	return gogChecksums, nil
}

func getGogFilenames(id string, rdx redux.Writeable, force bool) (map[string]string, error) {
	rcGogFilenames, err := getProductType(id, vangogh_integration.GogFilenames, rdx, force)
	if err != nil {
		return nil, err
	}
	defer rcGogFilenames.Close()

	var gogFilenames map[string]string
	if err = json.UnmarshalRead(rcGogFilenames, &gogFilenames); err != nil {
		return nil, err
	}

	return gogFilenames, nil
}

func getGogImages(id string, rdx redux.Writeable, force bool) (map[string][]string, error) {
	rcGogImages, err := getProductType(id, vangogh_integration.GogImages, rdx, force)
	if err != nil {
		return nil, err
	}
	defer rcGogImages.Close()

	var gogImages map[string][]string
	if err = json.UnmarshalRead(rcGogImages, &gogImages); err != nil {
		return nil, err
	}

	return gogImages, nil
}

func getGogDetails(id string, rdx redux.Writeable, force bool) (*gog_integration.Details, error) {
	rcDetails, err := getProductType(id, vangogh_integration.GogDetails, rdx, force)
	if err != nil {
		return nil, err
	}
	defer rcDetails.Close()

	return vangogh_integration.UnmarshalDetailsReadCloser(rcDetails)
}

func reduceGogDetails(id string, kvGogDetails kevlar.KeyValues, rdx redux.Writeable) error {

	rcGogDetails, err := kvGogDetails.Get(id)
	if err != nil {
		return err
	}

	defer rcGogDetails.Close()

	det, err := vangogh_integration.UnmarshalDetails(id, kvGogDetails)
	if err != nil {
		return err
	}

	propertyValues := make(map[string][]string)

	reductionProperties := []string{
		vangogh_integration.GogTitleProperty,
		vangogh_integration.GogOperatingSystemsProperty,
	}

	for _, property := range reductionProperties {

		var values []string

		switch property {
		case vangogh_integration.GogTitleProperty:
			values = []string{det.GetTitle()}
		case vangogh_integration.GogOperatingSystemsProperty:
			values, err = det.GetOperatingSystems()
			if err != nil {
				return err
			}
		}

		if len(values) == 1 && values[0] == "" {
			values = nil
		}

		if len(values) > 0 {
			propertyValues[property] = values
		}
	}

	for property, values := range propertyValues {
		if err = rdx.ReplaceValues(property, id, values...); err != nil {
			return err
		}
	}

	return nil
}
