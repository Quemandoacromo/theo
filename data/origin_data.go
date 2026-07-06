package data

import (
	"github.com/arelate/southern_light/egs_integration"
	"github.com/arelate/southern_light/gog_integration"
	"github.com/arelate/southern_light/steam_vdf"
)

type OriginData struct {
	GogDetails *gog_integration.Details
	//GogDownloadsList vangogh_integration.DownloadsList
	GogFilenames map[string]string
	AppInfoKv    steam_vdf.ValveDataFile
	CatalogItem  *egs_integration.CatalogItem
	GameManifest *egs_integration.GameManifest
	Manifest     *egs_integration.Manifest
}
