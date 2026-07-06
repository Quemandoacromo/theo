package cli

import (
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/arelate/southern_light/steamcmd"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/camino"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func SetupSteamCmdHandler(u *url.URL) error {

	q := u.Query()

	force := q.Has(vangogh_integration.UrlForceParameter)

	return SetupSteamCmd(force)
}

func SetupSteamCmd(force bool) error {

	currentOs := vangogh_integration.CurrentOs()

	ssca := nod.Begin("setting up SteamCMD for %s...", currentOs)
	defer ssca.Done()

	rdx, err := redux.NewWriter(vangogh_integration.AbsReduxDir(), data.VangoghProperties()...)
	if err != nil {
		return err
	}

	if err = vangoghValidateSessionToken(rdx); err != nil {
		return err
	}

	if err = downloadSteamCmdBinaries(currentOs, rdx, force); err != nil {
		return err
	}

	if err = unpackSteamCmdBinaries(currentOs, force); err != nil {
		return err
	}

	return nil
}

func downloadSteamCmdBinaries(operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable, force bool) error {

	dscba := nod.NewProgress(" downloading SteamCMD for %s...", operatingSystem)
	defer dscba.Done()

	steamCmdReleasesDir := camino.GetRel(vangogh_integration.Releases, vangogh_integration.Binaries)

	apiBinarySteamCmdPath := path.Join(data.ApiBinaryPath, operatingSystem.String(), steamcmd.Title)

	steamCmdBinaryUrl, err := data.VangoghUrl(apiBinarySteamCmdPath, nil, rdx)
	if err != nil {
		return err
	}

	dc := dolo.DefaultClient

	if token, ok := rdx.GetLastVal(data.VangoghSessionTokenProperty, data.VangoghSessionTokenProperty); ok && token != "" {
		dc.SetAuthorizationBearer(token)
	}

	relSteamCmdFilename := filepath.Base(steamcmd.Urls[operatingSystem])

	return dc.Download(steamCmdBinaryUrl, force, dscba, steamCmdReleasesDir, relSteamCmdFilename)
}

func unpackSteamCmdBinaries(operatingSystem vangogh_integration.OperatingSystem, force bool) error {

	uscba := nod.Begin(" unpacking SteamCMD for %s...", operatingSystem)
	defer uscba.Done()

	absSteamCmdBinaryPath, err := data.AbsSteamCmdBinPath(operatingSystem)
	if err != nil {
		return err
	}

	if _, err = os.Stat(absSteamCmdBinaryPath); err == nil && !force {
		uscba.EndWithResult("already unpacked")
		return nil
	}

	steamCmdReleasesDir := camino.GetRel(vangogh_integration.Releases, vangogh_integration.Binaries)
	absSteamCmdDownload := filepath.Join(steamCmdReleasesDir, filepath.Base(steamcmd.Urls[operatingSystem]))

	runtimesDir := camino.GetRel(vangogh_integration.Runtimes, vangogh_integration.Binaries)
	steamCmdRuntimesDir := filepath.Join(runtimesDir, steamcmd.Title)
	osSteamCmdBinariesDir := filepath.Join(steamCmdRuntimesDir, operatingSystem.String())

	return untar(absSteamCmdDownload, osSteamCmdBinariesDir)
}
