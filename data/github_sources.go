package data

import (
	_ "embed"
	"encoding/json"
	"github.com/arelate/southern_light/github_integration"
	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/pathways"
	"path/filepath"
	"strings"
)

const umuRunFilename = "umu-run"

type GitHubSource struct {
	OwnerRepo string
	Asset     string
	Unpack    []string
}

var GeProtonCustom = &GitHubSource{
	OwnerRepo: "GloriousEggroll/proton-ge-custom",
	Asset:     ".tar.gz",
}

var UmuLauncher = &GitHubSource{
	OwnerRepo: "Open-Wine-Components/umu-launcher",
	Asset:     "Zipapp.zip",
}

func (ghs *GitHubSource) GetLatestRelease() (*github_integration.GitHubRelease, error) {

	gitHubReleasesDir, err := pathways.GetAbsRelDir(GitHubReleases)
	if err != nil {
		return nil, err
	}

	kvGitHubReleases, err := kevlar.NewKeyValues(gitHubReleasesDir, kevlar.JsonExt)
	if err != nil {
		return nil, err
	}

	rcReleases, err := kvGitHubReleases.Get(ghs.OwnerRepo)
	if err != nil {
		return nil, err
	}

	var releases []github_integration.GitHubRelease
	if err := json.NewDecoder(rcReleases).Decode(&releases); err != nil {
		rcReleases.Close()
		return nil, err
	}

	if err := rcReleases.Close(); err != nil {
		return nil, err
	}

	var latestRelease *github_integration.GitHubRelease
	if len(releases) > 0 {
		latestRelease = &releases[0]
	}

	return latestRelease, nil
}

func (ghs *GitHubSource) GetAsset(release *github_integration.GitHubRelease) *github_integration.GitHubAsset {

	if len(release.Assets) == 1 {
		return &release.Assets[0]
	}

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, ghs.Asset) {
			return &asset
		}
	}

	return nil
}

func OsGitHubSources(os vangogh_integration.OperatingSystem) []*GitHubSource {
	switch os {
	case vangogh_integration.Linux:
		return []*GitHubSource{GeProtonCustom, UmuLauncher}
	default:
		return nil
	}
}

func GeProtonCustomLatestReleasePath() (string, error) {

	latestRelease, err := GeProtonCustom.GetLatestRelease()
	if err != nil {
		return "", err
	}

	geProtonCustomDir, err := GetAbsBinariesDir(GeProtonCustom, latestRelease)
	if err != nil {
		return "", err
	}

	// GE-Proton-custom archive is a packaged dir named after the latest release tag
	_, lastDir := filepath.Split(geProtonCustomDir)

	return filepath.Join(geProtonCustomDir, lastDir), nil
}

func UmuRunLatestReleasePath() (string, error) {

	latestRelease, err := UmuLauncher.GetLatestRelease()
	if err != nil {
		return "", err
	}

	absUmuBinDir, err := GetAbsBinariesDir(UmuLauncher, latestRelease)
	if err != nil {
		return "", err
	}

	return filepath.Join(absUmuBinDir, umuRunFilename), nil
}
