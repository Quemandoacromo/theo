package data

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/steamcmd"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/boggydigital/camino"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/redux"
)

func GetTitleProperty(id string, rdx redux.Readable) (string, error) {
	titleProperties := []string{
		vangogh_integration.GogTitleProperty,
		vangogh_integration.SteamTitleProperty,
		vangogh_integration.EgsTitleProperty,
	}

	if err := rdx.MustHave(titleProperties...); err != nil {
		return "", err
	}

	for _, tp := range titleProperties {
		if title, ok := rdx.GetLastVal(tp, id); ok && title != "" {
			return title, nil
		}
	}

	return "", errors.New("title not found for " + id)
}

func OsLangCode(operatingSystem vangogh_integration.OperatingSystem, langCode string) string {
	return strings.Join([]string{operatingSystem.String(), langCode}, "-")
}

func AppOsLangCode(id string, operatingSystem vangogh_integration.OperatingSystem, langCode string) string {
	return strings.Join([]string{id, operatingSystem.String(), langCode}, "-")
}

func AbsPrefixDir(id string, origin Origin, rdx redux.Readable) (string, error) {

	var prefixesDir string
	switch origin {
	case VangoghOrigin:
		prefixesDir = camino.GetRel(vangogh_integration.GogPrefixes, vangogh_integration.Prefixes)
	case SteamOrigin:
		prefixesDir = camino.GetRel(vangogh_integration.SteamPrefixes, vangogh_integration.Prefixes)
	case EpicGamesOrigin:
		prefixesDir = camino.GetRel(vangogh_integration.EgsPrefixes, vangogh_integration.Prefixes)
	default:
		return "", origin.ErrUnsupportedOrigin()
	}

	title, err := GetTitleProperty(id, rdx)
	if err != nil {
		return "", err
	}

	return filepath.Join(prefixesDir, camino.Sanitize(title)), nil
}

func AbsInventoryFilename(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) (string, error) {

	osLangInventoryDir := filepath.Join(camino.GetRel(vangogh_integration.Inventory, vangogh_integration.InstalledApps), OsLangCode(operatingSystem, langCode))

	title, err := GetTitleProperty(id, rdx)
	if err != nil {
		return "", err
	}

	return filepath.Join(osLangInventoryDir, camino.Sanitize(title)+kevlar.JsonExt), nil
}

func AbsSteamCmdBinPath(operatingSystem vangogh_integration.OperatingSystem) (string, error) {
	switch operatingSystem {
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		runtimesDir := camino.GetRel(vangogh_integration.Runtimes, vangogh_integration.Binaries)
		steamCmdRuntimesDir := filepath.Join(runtimesDir, steamcmd.Title)
		osSteamCmdBinariesDir := filepath.Join(steamCmdRuntimesDir, operatingSystem.String())
		return filepath.Join(osSteamCmdBinariesDir, vangogh_integration.SteamCmdOsBinary[operatingSystem]), nil
	default:
		return "", operatingSystem.ErrUnsupported()
	}
}

func AbsSteamAppInstallDir(steamAppId string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) (string, error) {

	if err := rdx.MustHave(vangogh_integration.SteamTitleProperty); err != nil {
		return "", err
	}

	var steamAppName string
	if san, ok := rdx.GetLastVal(vangogh_integration.SteamTitleProperty, steamAppId); ok && san != "" {
		steamAppName = san
	}

	if steamAppName == "" {
		return "", errors.New("Steam app name not found for " + steamAppId)
	}

	steamAppsDir := camino.GetRel(vangogh_integration.SteamApps, vangogh_integration.InstalledApps)

	return filepath.Join(steamAppsDir, operatingSystem.String(), camino.Sanitize(steamAppName)), nil
}

func AbsChunksDownloadDir(appName string, operatingSystem vangogh_integration.OperatingSystem) string {
	return filepath.Join(camino.GetAbs(vangogh_integration.Downloads), fmt.Sprintf("%s-%s", appName, operatingSystem))
}
