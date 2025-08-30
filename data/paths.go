package data

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
)

const theoDirname = "theo"

const (
	inventoryExt = ".txt"
)

func InitRootDir() (string, error) {
	udhd, err := UserDataHomeDir()
	if err != nil {
		return "", err
	}

	rootDir := filepath.Join(udhd, theoDirname)
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		if err := os.MkdirAll(rootDir, 0755); err != nil {
			return "", err
		}
	}

	for _, ad := range AllAbsDirs {
		absDir := filepath.Join(rootDir, string(ad))
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			if err := os.MkdirAll(absDir, 0755); err != nil {
				return "", err
			}
		}
	}

	for rd, ad := range RelToAbsDirs {
		absRelDir := filepath.Join(rootDir, string(ad), string(rd))
		if _, err := os.Stat(absRelDir); os.IsNotExist(err) {
			if err := os.MkdirAll(absRelDir, 0755); err != nil {
				return "", err
			}
		}
	}

	return filepath.Join(udhd, theoDirname), nil
}

const (
	Backups       pathways.AbsDir = "backups"
	Metadata      pathways.AbsDir = "metadata"
	Downloads     pathways.AbsDir = "downloads"
	Wine          pathways.AbsDir = "wine"
	InstalledApps pathways.AbsDir = "installed-apps"
	Logs          pathways.AbsDir = "logs"
)

const (
	Redux          pathways.RelDir = "_redux"
	ProductDetails pathways.RelDir = "_product-details"
	WineDownloads  pathways.RelDir = "_downloads"
	WineBinaries   pathways.RelDir = "_binaries"
	PrefixArchive  pathways.RelDir = "_prefix-archive"
	UmuConfigs     pathways.RelDir = "_umu-configs"
	Inventory      pathways.RelDir = "_inventory"
)

var RelToAbsDirs = map[pathways.RelDir]pathways.AbsDir{
	Redux:          Metadata,
	ProductDetails: Metadata,
	Inventory:      InstalledApps,
	PrefixArchive:  Backups,
	WineDownloads:  Wine,
	WineBinaries:   Wine,
	UmuConfigs:     Wine,
}

var AllAbsDirs = []pathways.AbsDir{
	Backups,
	Metadata,
	Downloads,
	Wine,
	InstalledApps,
	Logs,
}

func GetPrefixName(id string, rdx redux.Readable) (string, error) {
	if slug, ok := rdx.GetLastVal(vangogh_integration.SlugProperty, id); ok && slug != "" {
		return slug, nil
	} else {
		return "", errors.New("product slug is undefined: " + id)
	}
}

func OsLangCode(operatingSystem vangogh_integration.OperatingSystem, langCode string) string {
	return strings.Join([]string{operatingSystem.String(), langCode}, "-")
}

func GetAbsPrefixDir(id, langCode string, rdx redux.Readable) (string, error) {
	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return "", err
	}

	installedAppsDir, err := pathways.GetAbsDir(InstalledApps)
	if err != nil {
		return "", err
	}

	osLangInstalledAppsDir := filepath.Join(installedAppsDir, OsLangCode(vangogh_integration.Windows, langCode))

	prefixName, err := GetPrefixName(id, rdx)
	if err != nil {
		return "", err
	}

	return filepath.Join(osLangInstalledAppsDir, prefixName), nil
}

func GetAbsInventoryFilename(id, langCode string, operatingSystem vangogh_integration.OperatingSystem, rdx redux.Readable) (string, error) {
	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return "", err
	}

	inventoryDir, err := pathways.GetAbsRelDir(Inventory)
	if err != nil {
		return "", err
	}

	osLangInventoryDir := filepath.Join(inventoryDir, OsLangCode(operatingSystem, langCode))

	if slug, ok := rdx.GetLastVal(vangogh_integration.SlugProperty, id); ok && slug != "" {
		return filepath.Join(osLangInventoryDir, slug+inventoryExt), nil
	} else {
		return "", errors.New("product slug is undefined: " + id)
	}
}

func GetRelFilesModifiedAfter(absDir string, utcTime int64) ([]string, error) {
	files := make([]string, 0)

	if err := filepath.Walk(absDir, func(path string, info fs.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if info.ModTime().UTC().Unix() >= utcTime {
			relPath, err := filepath.Rel(absDir, path)
			if err != nil {
				return err
			}

			files = append(files, relPath)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}

func RelToUserDataHome(path string) (string, error) {
	udhd, err := UserDataHomeDir()
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(path, udhd) {
		return strings.Replace(path, udhd, "~Data", 1), nil
	} else {
		return path, nil
	}
}
