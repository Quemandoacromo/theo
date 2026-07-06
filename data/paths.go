package data

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/steamcmd"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/boggydigital/camino"
	"github.com/boggydigital/redux"
)

const theoDirname = "theo"

const (
	inventoryExt = ".json"
)

const (
	Backups camino.AbsDir = iota
	Binaries
	Downloads
	Logs
	Metadata
	InstalledApps
	Prefixes
)

var absDirNames = map[camino.AbsDir]string{
	Backups:       "backups",
	Binaries:      "binaries",
	Downloads:     "downloads",
	Logs:          "logs",
	Metadata:      "metadata",
	InstalledApps: "installed-apps",
	Prefixes:      "prefixes",
}

const (
	Redux camino.RelDir = iota
	GogDetails
	GogChecksums
	GogFilenames
	GogImages
	SteamAppInfo
	Cookies
	Tokens
	AvailableProducts
	EgsGameAssets
	EgsCatalogItems
	EgsGameManifests
	EgsManifests
	Temp
	Inventory
	GogApps
	SteamApps
	EgsApps
	UmuConfigs
	Releases
	Runtimes
	GogPrefixes
	SteamPrefixes
	EgsPrefixes
)

var relDirNames = map[camino.RelDir]string{
	Redux:             "_redux",
	GogDetails:        "gog-details",
	GogChecksums:      "gog-checksums",
	GogFilenames:      "gog-filenames",
	GogImages:         "gog-images",
	SteamAppInfo:      "steam-appinfo",
	Cookies:           "_cookies",
	Tokens:            "_tokens",
	AvailableProducts: "available-products",
	EgsGameAssets:     "egs-game-assets",
	EgsCatalogItems:   "egs-catalog-items",
	EgsGameManifests:  "egs-game-manifests",
	EgsManifests:      "egs-manifests",
	Inventory:         "_inventory",
	GogApps:           "gog-apps",
	SteamApps:         "steam-apps",
	EgsApps:           "egs-apps",
	Temp:              "_temp",
	Releases:          "releases",
	Runtimes:          "runtimes",
	GogPrefixes:       "gog-prefixes",
	SteamPrefixes:     "steam-prefixes",
	EgsPrefixes:       "egs-prefixes",
	UmuConfigs:        "_umu-configs",
}

var relAbsParents = map[camino.RelDir][]camino.AbsDir{
	Redux:             {Metadata},
	Cookies:           {Metadata},
	Tokens:            {Metadata},
	AvailableProducts: {Metadata},
	GogDetails:        {Metadata},
	GogChecksums:      {Metadata},
	GogFilenames:      {Metadata},
	GogImages:         {Metadata},
	SteamAppInfo:      {Metadata},
	EgsGameAssets:     {Metadata},
	EgsCatalogItems:   {Metadata},
	EgsGameManifests:  {Metadata},
	EgsManifests:      {Metadata},
	Temp:              {Downloads},
	Inventory:         {InstalledApps},
	GogApps:           {InstalledApps},
	SteamApps:         {InstalledApps},
	EgsApps:           {InstalledApps},
	UmuConfigs:        {InstalledApps},
	Releases:          {Binaries},
	Runtimes:          {Binaries},
	GogPrefixes:       {Prefixes},
	SteamPrefixes:     {Prefixes},
	EgsPrefixes:       {Prefixes},
}

var steamCmdBinary = map[vangogh_integration.OperatingSystem]string{
	vangogh_integration.MacOS: "steamcmd.sh",
	vangogh_integration.Linux: "steamcmd.sh",
}

func InitTheoCamino() error {
	udhd, err := UserDataHomeDir()
	if err != nil {
		return err
	}

	theoRootDir := filepath.Join(udhd, theoDirname)
	if _, err = os.Stat(theoRootDir); os.IsNotExist(err) {
		if err = os.MkdirAll(theoRootDir, camino.DefaultFileMode); err != nil {
			return err
		}
	}

	absDirPaths := make(map[camino.AbsDir]string)

	for ad, name := range absDirNames {
		absDirPaths[ad] = filepath.Join(theoRootDir, name)
	}

	return camino.Register(absDirPaths, relDirNames, relAbsParents)

}

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
		prefixesDir = camino.GetRel(GogPrefixes, Prefixes)
	case SteamOrigin:
		prefixesDir = camino.GetRel(SteamPrefixes, Prefixes)
	case EpicGamesOrigin:
		prefixesDir = camino.GetRel(EgsPrefixes, Prefixes)
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

	osLangInventoryDir := filepath.Join(camino.GetRel(Inventory, InstalledApps), OsLangCode(operatingSystem, langCode))

	title, err := GetTitleProperty(id, rdx)
	if err != nil {
		return "", err
	}

	return filepath.Join(osLangInventoryDir, camino.Sanitize(title)+inventoryExt), nil
}

func AbsSteamCmdBinPath(operatingSystem vangogh_integration.OperatingSystem) (string, error) {
	switch operatingSystem {
	case vangogh_integration.MacOS:
		fallthrough
	case vangogh_integration.Linux:
		runtimesDir := camino.GetRel(Runtimes, Binaries)
		steamCmdRuntimesDir := filepath.Join(runtimesDir, steamcmd.Title)
		osSteamCmdBinariesDir := filepath.Join(steamCmdRuntimesDir, operatingSystem.String())
		return filepath.Join(osSteamCmdBinariesDir, steamCmdBinary[operatingSystem]), nil
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

	steamAppsDir := camino.GetRel(SteamApps, InstalledApps)

	return filepath.Join(steamAppsDir, operatingSystem.String(), camino.Sanitize(steamAppName)), nil
}

func AbsChunksDownloadDir(appName string, operatingSystem vangogh_integration.OperatingSystem) string {
	return filepath.Join(camino.GetAbs(Downloads), fmt.Sprintf("%s-%s", appName, operatingSystem))
}

func AbsReduxDir() string {
	return camino.GetRel(Redux, Metadata)
}
