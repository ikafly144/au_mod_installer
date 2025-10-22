package aumgr

import (
	"encoding/json"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

const epicManifestID = "4AD6AD0447626FA05A0648B2A5D8C66A"

func getEpicManifest() (Manifest, error) {
	pd, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(pd, "Epic", "EpicGamesLauncher", "Data", "Manifests", epicManifestID+".item")
	osInfo, err := os.Stat(path)
	if err != nil || osInfo.IsDir() {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var manifest epicManifest
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

type epicManifest struct {
	FormatVersion               int      `json:"FormatVersion"`
	BIsIncompleteInstall        bool     `json:"bIsIncompleteInstall"`
	LaunchCommand               string   `json:"LaunchCommand"`
	LaunchExecutable            string   `json:"LaunchExecutable"`
	ManifestLocation            string   `json:"ManifestLocation"`
	ManifestHash                string   `json:"ManifestHash"`
	BIsApplication              bool     `json:"bIsApplication"`
	BIsExecutable               bool     `json:"bIsExecutable"`
	BIsManaged                  bool     `json:"bIsManaged"`
	BNeedsValidation            bool     `json:"bNeedsValidation"`
	BRequiresAuth               bool     `json:"bRequiresAuth"`
	BAllowMultipleInstances     bool     `json:"bAllowMultipleInstances"`
	BCanRunOffline              bool     `json:"bCanRunOffline"`
	BAllowURICmdArgs            bool     `json:"bAllowUriCmdArgs"`
	BLaunchElevated             bool     `json:"bLaunchElevated"`
	BaseURLs                    []string `json:"BaseURLs"`
	BuildLabel                  string   `json:"BuildLabel"`
	AppCategories               []string `json:"AppCategories"`
	ChunkDbs                    []any    `json:"ChunkDbs"`
	CompatibleApps              []any    `json:"CompatibleApps"`
	DisplayName                 string   `json:"DisplayName"`
	InstallationGUID            string   `json:"InstallationGuid"`
	InstallLocation             string   `json:"InstallLocation"`
	InstallSessionID            string   `json:"InstallSessionId"`
	InstallTags                 []any    `json:"InstallTags"`
	InstallComponents           []any    `json:"InstallComponents"`
	HostInstallationGUID        string   `json:"HostInstallationGuid"`
	PrereqIds                   []any    `json:"PrereqIds"`
	PrereqSHA1Hash              string   `json:"PrereqSHA1Hash"`
	LastPrereqSucceededSHA1Hash string   `json:"LastPrereqSucceededSHA1Hash"`
	StagingLocation             string   `json:"StagingLocation"`
	TechnicalType               string   `json:"TechnicalType"`
	VaultThumbnailURL           string   `json:"VaultThumbnailUrl"`
	VaultTitleText              string   `json:"VaultTitleText"`
	InstallSize                 int      `json:"InstallSize"`
	MainWindowProcessName       string   `json:"MainWindowProcessName"`
	ProcessNames                []any    `json:"ProcessNames"`
	BackgroundProcessNames      []any    `json:"BackgroundProcessNames"`
	IgnoredProcessNames         []any    `json:"IgnoredProcessNames"`
	DlcProcessNames             []any    `json:"DlcProcessNames"`
	MandatoryAppFolderName      string   `json:"MandatoryAppFolderName"`
	OwnershipToken              string   `json:"OwnershipToken"`
	SidecarConfigRevision       int      `json:"SidecarConfigRevision"`
	CatalogNamespace            string   `json:"CatalogNamespace"`
	CatalogItemID               string   `json:"CatalogItemId"`
	AppName                     string   `json:"AppName"`
	AppVersionString            string   `json:"AppVersionString"`
	MainGameCatalogNamespace    string   `json:"MainGameCatalogNamespace"`
	MainGameCatalogItemID       string   `json:"MainGameCatalogItemId"`
	MainGameAppName             string   `json:"MainGameAppName"`
	AllowedURIEnvVars           []any    `json:"AllowedUriEnvVars"`
}

func (e epicManifest) GetVersion() string {
	return e.AppVersionString
}
