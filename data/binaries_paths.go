package data

import (
	_ "embed"
	"errors"
	"os"
	"path/filepath"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/southern_light/wine_integration"
	"github.com/boggydigital/camino"
	"github.com/boggydigital/redux"
)

const umuRunBinaryFn = "umu/umu-run"

func UmuRunLatestReleasePath(rdx redux.Readable) (string, error) {
	return githubLatestReleasePath(umuRunBinaryFn, wine_integration.UmuLauncher, rdx)
}

func ProtonLatestReleasePath(runtime string, rdx redux.Readable) (string, error) {

	if runtime == "" {
		runtime = wine_integration.ProtonGeCustom
	}

	return githubLatestReleasePath("", runtime, rdx)
}

func InnoextractLatestReleasePath(relBinaryPath string, rdx redux.Readable) (string, error) {
	return githubLatestReleasePath(relBinaryPath, wine_integration.Innoextract, rdx)
}

func githubLatestReleasePath(relBinPath string, repo string, rdx redux.Readable) (string, error) {

	if err := rdx.MustHave(WineBinariesVersionsProperty); err != nil {
		return "", err
	}

	var latestRelease string
	if wbvp, ok := rdx.GetLastVal(WineBinariesVersionsProperty, repo); ok {
		latestRelease = wbvp
	}

	if latestRelease == "" {
		return "", errors.New(repo + " latest version not found, please run setup-wine")
	}

	absBinPath := filepath.Join(camino.GetRel(vangogh_integration.Runtimes, vangogh_integration.Binaries), camino.Sanitize(repo), latestRelease)
	if relBinPath != "" {
		absBinPath = filepath.Join(absBinPath, relBinPath)
	}
	if _, err := os.Stat(absBinPath); err == nil {
		return absBinPath, nil
	}

	return "", os.ErrNotExist
}
