package cli

import (
	"bufio"
	"bytes"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/southern_light/wine_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/camino"
	"github.com/boggydigital/dolo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func SetupWineHandler(u *url.URL) error {

	q := u.Query()

	force := q.Has(vangogh_integration.UrlForceParameter)

	return SetupWine(force)
}

func SetupWine(force bool) error {

	start := time.Now()

	currentOs := vangogh_integration.CurrentOs()

	if currentOs == vangogh_integration.Windows {
		err := errors.New("WINE is not required on Windows")
		return err
	}

	uwa := nod.Begin("setting up WINE for %s...", currentOs)
	defer uwa.Done()

	properties := append(data.VangoghProperties(), data.WineBinariesVersionsProperty)

	rdx, err := redux.NewWriter(vangogh_integration.AbsReduxDir(), properties...)
	if err != nil {
		return err
	}

	if err = vangoghValidateSessionToken(rdx); err != nil {
		return err
	}

	wbd, err := getWineBinariesVersions(rdx)
	if err != nil {
		return err
	}

	if err = downloadWineBinaries(wbd, currentOs, rdx, force); err != nil {
		return err
	}

	if err = validateWineBinaries(wbd, currentOs, start, force); err != nil {
		return err
	}

	if err = pinWineBinariesVersions(wbd, rdx); err != nil {
		return err
	}

	if err = cleanupDownloadedWineBinaries(wbd, currentOs); err != nil {
		return err
	}

	if err = unpackWineBinaries(wbd, currentOs, force); err != nil {
		return err
	}

	if err = cleanupUnpackedWineBinaries(wbd, currentOs); err != nil {
		return err
	}

	return nil
}

func getWineBinariesVersions(rdx redux.Readable) ([]vangogh_integration.WineBinaryDetails, error) {

	gwbva := nod.Begin("getting WINE binaries versions...")
	defer gwbva.Done()

	if err := rdx.MustHave(data.VangoghProperties()...); err != nil {
		return nil, err
	}

	req, err := data.VangoghApiRequest(http.MethodGet, data.ApiBinariesVersionsPath, nil, rdx)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, errors.New(resp.Status)
	}

	var wbd []vangogh_integration.WineBinaryDetails

	if err = json.UnmarshalRead(resp.Body, &wbd); err != nil {
		return nil, err
	}

	return wbd, nil
}

func downloadWineBinaries(wbd []vangogh_integration.WineBinaryDetails,
	operatingSystem vangogh_integration.OperatingSystem,
	rdx redux.Readable,
	force bool) error {

	dwba := nod.Begin("downloading WINE binaries...")
	defer dwba.Done()

	for _, wineBinary := range wbd {
		if wineBinary.OS != operatingSystem && wineBinary.OS != vangogh_integration.Windows {
			continue
		}

		if err := downloadWineBinary(&wineBinary, rdx, force); err != nil {
			return err
		}
	}

	return nil
}

func downloadWineBinary(binary *vangogh_integration.WineBinaryDetails, rdx redux.Readable, force bool) error {

	dwba := nod.NewProgress(" - %s %s...", binary.Title, binary.Version)
	defer dwba.Done()

	properties := append(data.VangoghProperties(), data.WineBinariesVersionsProperty)

	if err := rdx.MustHave(properties...); err != nil {
		return err
	}

	binariesReleasesDir := camino.GetRel(vangogh_integration.Releases, vangogh_integration.Binaries)

	if currentVersion, ok := rdx.GetLastVal(data.WineBinariesVersionsProperty, binary.Title); ok && binary.Version == currentVersion && !force {
		dwba.EndWithResult("latest version already available")
		return nil
	}

	apiBinaryPath := path.Join(data.ApiBinaryPath, binary.OS.String(), binary.Title)

	wineBinaryUrl, err := data.VangoghUrl(apiBinaryPath, nil, rdx)
	if err != nil {
		return err
	}

	dc := dolo.DefaultClient

	if token, ok := rdx.GetLastVal(data.VangoghSessionTokenProperty, data.VangoghSessionTokenProperty); ok && token != "" {
		dc.SetAuthorizationBearer(token)
	}

	return dc.Download(wineBinaryUrl, force, dwba, binariesReleasesDir, binary.Filename)
}

func validateWineBinaries(wbd []vangogh_integration.WineBinaryDetails, operatingSystem vangogh_integration.OperatingSystem, since time.Time, force bool) error {

	vwba := nod.NewProgress("validating WINE binaries...")
	defer vwba.Done()

	binariesReleasesDir := camino.GetRel(vangogh_integration.Releases, vangogh_integration.Binaries)

	for _, wineBinary := range wbd {
		if wineBinary.OS != operatingSystem && wineBinary.OS != vangogh_integration.Windows {
			continue
		}

		if err := wine_integration.ValidateWineBinary(&wineBinary, binariesReleasesDir, since, force); err != nil {
			return err
		}
	}

	return nil
}

func pinWineBinariesVersions(wbd []vangogh_integration.WineBinaryDetails, rdx redux.Writeable) error {

	pwbva := nod.Begin("pinning WINE binaries versions...")
	defer pwbva.Done()

	if err := rdx.MustHave(data.WineBinariesVersionsProperty); err != nil {
		return err
	}

	wineBinariesVersions := make(map[string][]string)

	for _, wineBinary := range wbd {
		wineBinariesVersions[wineBinary.Title] = []string{wineBinary.Version}
	}

	return rdx.BatchReplaceValues(data.WineBinariesVersionsProperty, wineBinariesVersions)
}

func cleanupDownloadedWineBinaries(wbd []vangogh_integration.WineBinaryDetails, operatingSystem vangogh_integration.OperatingSystem) error {

	cdwba := nod.NewProgress("cleaning up downloaded WINE binaries...")
	defer cdwba.Done()

	expectedFiles := make([]string, 0, len(wbd))
	for _, wineBinary := range wbd {
		if wineBinary.OS != operatingSystem && wineBinary.OS != vangogh_integration.Windows {
			continue
		}
		expectedFiles = append(expectedFiles, wineBinary.Filename)
	}

	binariesReleasesDir := camino.GetRel(vangogh_integration.Releases, vangogh_integration.Binaries)

	wineDownloadsDir, err := os.Open(binariesReleasesDir)
	if err != nil {
		return err
	}

	defer wineDownloadsDir.Close()

	actualFiles, err := wineDownloadsDir.Readdirnames(-1)
	if err != nil {
		return err
	}

	unexpectedFiles := make([]string, 0)

	for _, af := range actualFiles {
		if strings.HasPrefix(af, ".") {
			continue
		}
		if !slices.Contains(expectedFiles, af) {
			unexpectedFiles = append(unexpectedFiles, af)
		}
	}

	if len(unexpectedFiles) == 0 {
		cdwba.EndWithResult("already clean")
		return nil
	}

	cdwba.TotalInt(len(unexpectedFiles))

	for _, uf := range unexpectedFiles {
		absUnexpectedFile := filepath.Join(binariesReleasesDir, uf)
		if err = os.Remove(absUnexpectedFile); err != nil {
			return err
		}
		cdwba.Increment()
	}

	return nil
}

func unpackWineBinaries(wbd []vangogh_integration.WineBinaryDetails,
	operatingSystem vangogh_integration.OperatingSystem,
	force bool) error {

	uwba := nod.Begin("unpacking WINE binaries...")
	defer uwba.Done()

	binariesReleasesDir := camino.GetRel(vangogh_integration.Releases, vangogh_integration.Binaries)
	binariesRuntimesDir := camino.GetRel(vangogh_integration.Runtimes, vangogh_integration.Binaries)

	for _, wineBinary := range wbd {
		if wineBinary.OS != operatingSystem {
			continue
		}

		srcPath := filepath.Join(binariesReleasesDir, wineBinary.Filename)
		dstPath := filepath.Join(binariesRuntimesDir, camino.Sanitize(wineBinary.Title), wineBinary.Version)

		if _, err := os.Stat(dstPath); err == nil && !force {
			continue
		}

		wba := nod.Begin(" - %s...", wineBinary.Title)

		tarFiles, err := tarTf(srcPath)
		if err != nil {
			return err
		}

		var scOption string

		// TODO: refactor this

		ext := filepath.Ext(wineBinary.Filename)
		switch ext {
		case ".xz":
			fallthrough
		case ".gz":
			if filepath.Ext(strings.TrimSuffix(wineBinary.Filename, ext)) == ".tar" {
				switch vangogh_integration.CurrentOs() {
				case vangogh_integration.Linux:
					scOption = "--strip-components=1"
					if len(tarFiles) > 0 {
						if strings.HasPrefix(tarFiles[0], "./") {
							scOption = "--strip-components=2"
						}
					}
				default:
					// do nothing
				}
			}
		default:
			// do nothing
		}

		if err = untar(srcPath, dstPath, scOption); err != nil {
			return err
		}

		wba.Done()
	}

	return nil
}

func cleanupUnpackedWineBinaries(wbd []vangogh_integration.WineBinaryDetails,
	operatingSystem vangogh_integration.OperatingSystem) error {

	cuwba := nod.NewProgress("cleaning up unpacked WINE binaries...")
	defer cuwba.Done()

	runtimesDir := camino.GetRel(vangogh_integration.Runtimes, vangogh_integration.Binaries)

	absExpectedDirs := make([]string, 0)
	absActualDirs := make([]string, 0)

	for _, wineBinary := range wbd {
		if wineBinary.OS != operatingSystem {
			continue
		}

		absTitleDir := filepath.Join(runtimesDir, camino.Sanitize(wineBinary.Title))

		absLatestVersionDir := filepath.Join(absTitleDir, wineBinary.Version)
		absExpectedDirs = append(absExpectedDirs, absLatestVersionDir)

		titleDir, err := os.Open(absTitleDir)
		if err != nil {
			return err
		}

		var filenames []string
		filenames, err = titleDir.Readdirnames(-1)
		if err != nil {
			if err = titleDir.Close(); err != nil {
				return err
			}
			return err
		}

		for _, fn := range filenames {
			absActualDirs = append(absActualDirs, filepath.Join(absTitleDir, fn))
		}

		if err = titleDir.Close(); err != nil {
			return err
		}
	}

	absUnexpectedDirs := make([]string, 0)

	for _, aad := range absActualDirs {
		if !slices.Contains(absExpectedDirs, aad) {
			absUnexpectedDirs = append(absUnexpectedDirs, aad)
		}
	}

	if len(absUnexpectedDirs) == 0 {
		cuwba.EndWithResult("already clean")
		return nil
	}

	cuwba.TotalInt(len(absUnexpectedDirs))

	for _, aud := range absUnexpectedDirs {
		if err := os.RemoveAll(aud); err != nil {
			return err
		}
		cuwba.Increment()
	}

	return nil
}

func untar(srcPath, dstPath string, options ...string) error {

	tarPath, err := exec.LookPath("tar")
	if err != nil {
		return err
	}

	args := []string{"-xf", srcPath}

	if dstPath != "" {
		if _, err = os.Stat(dstPath); err != nil {
			if err = os.MkdirAll(dstPath, camino.DefaultFileMode); err != nil {
				return err
			}
		}
		args = append(args, "-C", dstPath)
	}

	for _, opt := range options {
		if opt != "" {
			args = append(args, opt)
		}
	}

	cmd := exec.Command(tarPath, args...)

	return cmd.Run()
}

func unzip(srcPath, dstPath string, options ...string) error {

	unzipPath, err := exec.LookPath("unzip")
	if err != nil {
		return err
	}

	args := []string{srcPath}

	if dstPath != "" {
		if _, err = os.Stat(dstPath); err != nil {
			if err = os.MkdirAll(dstPath, camino.DefaultFileMode); err != nil {
				return err
			}
		}
		args = append(args, "-d", dstPath)
	}

	for _, opt := range options {
		if opt != "" {
			args = append(args, opt)
		}
	}

	cmd := exec.Command(unzipPath, args...)

	return cmd.Run()
}

func tarTf(srcPath string) ([]string, error) {

	tarPath, err := exec.LookPath("tar")
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)

	cmd := exec.Command(tarPath, "tf", srcPath)
	cmd.Stdout = buf

	if err = cmd.Run(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(buf)

	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
