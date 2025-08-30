package main

import (
	"bytes"
	_ "embed"
	"log"
	"net/url"
	"os"

	"github.com/arelate/theo/cli"
	"github.com/arelate/theo/clo_delegates"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/clo"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
)

var (
	//go:embed "cli-commands.txt"
	cliCommands []byte
	//go:embed "cli-help.txt"
	cliHelp []byte
)

const debugParam = "debug"

func main() {

	nod.EnableStdOutPresenter()

	tsa := nod.Begin("theo is complementing vangogh experience")
	defer tsa.Done()

	theoRootDir, err := data.InitRootDir()
	if err != nil {
		log.Fatalln(err)
	}

	if err = pathways.Setup("",
		theoRootDir,
		data.RelToAbsDirs,
		data.AllAbsDirs...); err != nil {
		log.Fatalln(err)
	}

	defs, err := clo.Load(
		bytes.NewBuffer(cliCommands),
		bytes.NewBuffer(cliHelp),
		clo_delegates.FuncMap)
	if err != nil {
		log.Fatalln(err)
	}

	clo.HandleFuncs(map[string]clo.Handler{
		"backup-metadata":  cli.BackupMetadataHandler,
		"download":         cli.DownloadHandler,
		"install":          cli.InstallHandler,
		"list":             cli.ListHandler,
		"prefix":           cli.PrefixHandler,
		"remove-downloads": cli.RemoveDownloadsHandler,
		"reveal":           cli.RevealHandler,
		"run":              cli.RunHandler,
		"setup-server":     cli.SetupServerHandler,
		"setup-wine":       cli.SetupWineHandler,
		"steam-shortcut":   cli.SteamShortcutHandler,
		"uninstall":        cli.UninstallHandler,
		"update":           cli.UpdateHandler,
		"validate":         cli.ValidateHandler,
		"version":          cli.VersionHandler,
	})

	if err = defs.AssertCommandsHaveHandlers(); err != nil {
		log.Fatalln(err)
	}

	var u *url.URL
	if u, err = defs.Parse(os.Args[1:]); err != nil {
		log.Fatalln(err)
	}

	if q := u.Query(); q.Has(debugParam) {
		absLogsDir, err := pathways.GetAbsDir(data.Logs)
		if err != nil {
			log.Fatalln(err)
		}
		logger, err := nod.EnableFileLogger(u.Path, absLogsDir)
		if err != nil {
			log.Fatalln(err)
		}
		defer logger.Close()
	}

	if err = defs.Serve(u); err != nil {
		tsa.Error(err)
		log.Fatalln(err)
	}
}
