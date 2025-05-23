package cli

import (
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func linuxProtonRun(id, langCode string, rdx redux.Readable, env []string, verbose, force bool, exePath, pwdPath string, arg ...string) error {

	_, exeFilename := filepath.Split(exePath)

	lwra := nod.Begin(" running %s with WINE, please wait...", exeFilename)
	defer lwra.Done()

	if err := rdx.MustHave(
		vangogh_integration.SlugProperty,
		vangogh_integration.SteamAppIdProperty); err != nil {
		return err
	}

	if verbose && len(env) > 0 {
		pea := nod.Begin(" env:")
		pea.EndWithResult(strings.Join(env, " "))
	}

	absUmuRunPath, err := data.UmuRunLatestReleasePath()
	if err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	absProtonPath, err := data.UmuProtonLatestReleasePath()
	if err != nil {
		return err
	}

	umuCfg := &UmuConfig{
		GogId:   id,
		Prefix:  absPrefixDir,
		Proton:  absProtonPath,
		ExePath: exePath,
		Args:    arg,
	}

	if steamAppId, ok := rdx.GetLastVal(vangogh_integration.SteamAppIdProperty, id); ok && steamAppId != "" {
		umuCfg.SteamAppId = steamAppId
	}

	absUmuConfigPath, err := createUmuConfig(umuCfg, force)
	if err != nil {
		return err
	}

	cmd := exec.Command(absUmuRunPath, "--config", absUmuConfigPath)

	if pwdPath != "" {
		cmd.Dir = pwdPath
	}

	cmd.Env = append(os.Environ(), env...)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

func linuxInitPrefix(id, langCode string, rdx redux.Readable, _ bool) error {
	lipa := nod.Begin(" initializing prefix...")
	defer lipa.Done()

	if err := rdx.MustHave(vangogh_integration.SlugProperty); err != nil {
		return err
	}

	absPrefixDir, err := data.GetAbsPrefixDir(id, langCode, rdx)
	if err != nil {
		return err
	}

	return os.MkdirAll(absPrefixDir, 0755)
}
