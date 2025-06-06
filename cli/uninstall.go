package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"net/url"
	"path/filepath"
)

func UninstallHandler(u *url.URL) error {

	q := u.Query()

	ids := Ids(u)

	_, langCodes, _ := OsLangCodeDownloadType(u)

	langCode := defaultLangCode
	if len(langCodes) > 0 {
		langCode = langCodes[0]
	}

	force := q.Has("force")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.AllProperties()...)
	if err != nil {
		return err
	}

	return Uninstall(langCode, rdx, force, ids...)
}

func Uninstall(langCode string, rdx redux.Writeable, force bool, ids ...string) error {

	ua := nod.NewProgress("uninstalling products...")
	defer ua.Done()

	if !force {
		ua.EndWithResult("this operation requires force flag")
		return nil
	}

	installedDetailsDir, err := pathways.GetAbsRelDir(data.InstalledDetails)
	if err != nil {
		return err
	}

	osLangInstalledDetailsDir := filepath.Join(installedDetailsDir, data.OsLangCode(data.CurrentOs(), langCode))

	kvOsLangInstalledDetails, err := kevlar.New(osLangInstalledDetailsDir, kevlar.JsonExt)
	if err != nil {
		return err
	}

	var flattened bool
	if ids, flattened, err = gameProductTypesFlatMap(rdx, force, ids...); err != nil {
		return err
	} else if flattened {
		ua.EndWithResult("uninstalling PACK included games")
		return Uninstall(langCode, rdx, force, ids...)
	}

	ua.TotalInt(len(ids))

	for _, id := range ids {
		if err = currentOsUninstallProduct(id, langCode, rdx); err != nil {
			return err
		}

		if err = kvOsLangInstalledDetails.Cut(id); err != nil {
			return err
		}

		ua.Increment()
	}

	if err = unpinInstallParameters(data.CurrentOs(), langCode, rdx, ids...); err != nil {
		return err
	}

	if err = RemoveSteamShortcut(rdx, ids...); err != nil {
		return err
	}

	return nil

}

func currentOsUninstallProduct(id, langCode string, rdx redux.Readable) error {
	currentOs := data.CurrentOs()
	switch currentOs {
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		if err := nixUninstallProduct(id, langCode, currentOs, rdx); err != nil {
			return err
		}
	case vangogh_integration.Windows:
		if err := windowsUninstallProduct(id, langCode, rdx); err != nil {
			return err
		}
	default:
		return currentOs.ErrUnsupported()
	}
	return nil
}
