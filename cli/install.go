package cli

import (
	"errors"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	pkgExt = ".pkg"
	exeExt = ".exe"
	shExt  = ".sh"
)

const (
	relPayloadPath = "package.pkg/Scripts/payload"
)

func InstallHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)
	_, langCodes, downloadTypes := OsLangCodeDownloadType(u)

	langCode := defaultLangCode
	if len(langCodes) > 0 {
		langCode = langCodes[0]
	}

	ip := &installParameters{
		operatingSystem: data.CurrentOs(),
		langCode:        langCode,
		downloadTypes:   downloadTypes,
		keepDownloads:   q.Has("keep-downloads"),
		noSteamShortcut: q.Has("no-steam-shortcut"),
		reveal:          q.Has("reveal"),
		force:           q.Has("force"),
	}

	return Install(ip, ids...)
}

func Install(ip *installParameters, ids ...string) error {

	ia := nod.Begin("installing products...")
	defer ia.Done()

	currentOs := []vangogh_integration.OperatingSystem{ip.operatingSystem}
	langCodes := []string{ip.langCode}

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	vangogh_integration.PrintParams(ids, currentOs, langCodes, ip.downloadTypes, true)

	var flattened bool
	if ids, flattened, err = gameProductTypesFlatMap(rdx, ip.force, ids...); err != nil {
		return err
	} else if flattened {
		ia.EndWithResult("installing PACK included games")
		return Install(ip, ids...)
	}

	supported, err := filterNotSupported(ip.langCode, rdx, ip.force, ids...)
	if err != nil {
		return err
	}

	if len(supported) > 0 {
		ids = supported
	} else {
		ia.EndWithResult("requested products are not supported on %s", data.CurrentOs())
		return nil
	}

	notInstalled, err := filterNotInstalled(data.CurrentOs(), ip.langCode, ids...)
	if err != nil {
		return err
	}

	if len(notInstalled) > 0 {
		if !ip.force {
			ids = notInstalled
		}
	} else if !ip.force {
		ia.EndWithResult("all requested products are already installed")
		return nil
	}

	if err = BackupMetadata(); err != nil {
		return err
	}

	if err = Download(currentOs, langCodes, ip.downloadTypes, nil, rdx, ip.force, ids...); err != nil {
		return err
	}

	if err = Validate(currentOs, langCodes, ip.downloadTypes, nil, rdx, ids...); err != nil {
		return err
	}

	for _, id := range ids {
		if err = currentOsInstallProduct(id, ip.langCode, ip.downloadTypes, rdx, ip.force); err != nil {
			return err
		}
	}

	if !ip.noSteamShortcut {
		for _, id := range ids {
			if err = AddSteamShortcut(id, ip.operatingSystem, ip.langCode, rdx, ip.force); err != nil {
				return err
			}
		}
	}

	if !ip.keepDownloads {
		if err = RemoveDownloads(currentOs, langCodes, ip.downloadTypes, rdx, ip.force, ids...); err != nil {
			return err
		}
	}

	if err = pinInstalledDetails(currentOs, ip.langCode, ip.force, ids...); err != nil {
		return err
	}

	if err = pinInstallParameters(ip, rdx, ids...); err != nil {
		return err
	}

	if err = setInstallDates(rdx, ids...); err != nil {
		return err
	}

	if ip.reveal {
		if err = RevealInstalled(ip.langCode, ids...); err != nil {
			return err
		}
	}

	return nil
}

func setInstallDates(rdx redux.Writeable, ids ...string) error {

	if err := rdx.MustHave(data.InstallDateProperty); err != nil {
		return err
	}

	installDates := make(map[string][]string)
	now := time.Now().UTC().Format(time.RFC3339)

	for _, id := range ids {
		installDates[id] = []string{now}
	}
	return rdx.BatchReplaceValues(data.InstallDateProperty, installDates)
}

func filterNotInstalled(operatingSystem vangogh_integration.OperatingSystem, langCode string, ids ...string) ([]string, error) {

	fnia := nod.Begin(" checking existing installations...")
	defer fnia.Done()

	notInstalled := make([]string, 0, len(ids))

	installedDetailsDir, err := pathways.GetAbsRelDir(data.InstalledDetails)
	if err != nil {
		return nil, err
	}

	osLangInstalledDetailsDir := filepath.Join(installedDetailsDir, data.OsLangCode(operatingSystem, langCode))

	kvOsLangInstalledDetails, err := kevlar.New(osLangInstalledDetailsDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	for _, id := range ids {

		if kvOsLangInstalledDetails.Has(id) {
			continue
		}

		notInstalled = append(notInstalled, id)
	}

	if len(notInstalled) == 0 {
		fnia.EndWithResult("all products have existing installations: %s", strings.Join(ids, ","))
	} else {
		fnia.EndWithResult(
			"%d product(s) require installation: %s",
			len(notInstalled),
			strings.Join(notInstalled, ","))
	}

	return notInstalled, nil
}

func filterNotSupported(langCode string, rdx redux.Writeable, force bool, ids ...string) ([]string, error) {

	fnsa := nod.NewProgress(" checking operating systems support...")
	defer fnsa.Done()

	fnsa.TotalInt(len(ids))

	supported := make([]string, 0, len(ids))

	for _, id := range ids {

		productDetails, err := GetProductDetails(id, rdx, force)
		if err != nil {
			return nil, err
		}

		dls := productDetails.DownloadLinks.
			FilterOperatingSystems(data.CurrentOs()).
			FilterLanguageCodes(langCode).
			FilterDownloadTypes(vangogh_integration.Installer)

		if len(dls) > 0 {
			supported = append(supported, id)
		}

		fnsa.Increment()
	}

	return supported, nil
}

func currentOsInstallProduct(id string, langCode string, downloadTypes []vangogh_integration.DownloadType, rdx redux.Writeable, force bool) error {

	coipa := nod.Begin(" installing %s on %s...", id, data.CurrentOs())
	defer coipa.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	productDetails, err := GetProductDetails(id, rdx, force)
	if err != nil {
		return err
	}

	dls := productDetails.DownloadLinks.
		FilterOperatingSystems(data.CurrentOs()).
		FilterLanguageCodes(langCode).
		FilterDownloadTypes(downloadTypes...)

	if len(dls) == 0 {
		coipa.EndWithResult("no links are matching operating params")
		return nil
	}

	installedAppsDir, err := pathways.GetAbsDir(data.InstalledApps)
	if err != nil {
		return err
	}

	if err = hasFreeSpaceForProduct(productDetails, installedAppsDir,
		[]vangogh_integration.OperatingSystem{data.CurrentOs()}, []string{langCode}, downloadTypes, nil, force); err != nil {
		return err
	}

	installerExts := []string{pkgExt, shExt, exeExt}

	for _, link := range dls {

		if !slices.Contains(installerExts, filepath.Ext(link.LocalFilename)) {
			continue
		}

		switch link.OperatingSystem {
		case vangogh_integration.MacOS:
			if err = macOsInstallProduct(id, productDetails, &link, rdx, force); err != nil {
				return err
			}
		case vangogh_integration.Linux:
			if err = linuxInstallProduct(id, productDetails, &link, rdx); err != nil {
				return err
			}
		case vangogh_integration.Windows:
			if err = windowsInstallProduct(id, productDetails, &link, rdx); err != nil {
				return err
			}
		default:
			return errors.New("unknown os" + link.OperatingSystem.String())
		}
	}
	return nil
}

func gameProductTypesFlatMap(rdx redux.Writeable, force bool, ids ...string) ([]string, bool, error) {

	productsDetails, err := GetProductsDetails(rdx, force, ids...)
	if err != nil {
		return nil, false, err
	}

	if dlcs := productsDetails.Filter(vangogh_integration.DlcProductType); len(dlcs) > 0 {
		da := nod.Begin(" skipping DLC products " + strings.Join(dlcs.Ids(), ","))
		da.Done()
		productsDetails = productsDetails.Filter(vangogh_integration.GameProductType, vangogh_integration.PackProductType)
	}

	if gameProductsDetails := productsDetails.Filter(vangogh_integration.GameProductType); len(gameProductsDetails) == len(productsDetails) {
		return productsDetails.Ids(), false, nil
	} else {
		return productsDetails.GameAndIncludedGamesIds(), true, nil
	}
}
