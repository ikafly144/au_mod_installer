package aumgr

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/ikafly144/au_mod_installer/pkg/acf"
)

const steamAppID = "945360"

func getSteamManifest(gameDir string) (Manifest, error) {
	path := filepath.Join(gameDir, "..", "..", "appmanifest_"+steamAppID+".acf")
	osInfo, err := os.Stat(path)
	if err != nil || osInfo.IsDir() {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	acfData, err := acf.FromString(string(buf))
	if err != nil {
		return nil, err
	}
	jsonBuf := new(bytes.Buffer)
	if err := json.NewEncoder(jsonBuf).Encode(acfData); err != nil {
		return nil, err
	}
	var manifest steamManifest
	if err := json.NewDecoder(jsonBuf).Decode(&manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

type steamManifest struct {
	AppState AppState `json:"AppState"`
}

func (s steamManifest) GetVersion() string {
	return s.AppState.Buildid
}

type AppState struct {
	AllowOtherDownloadsWhileRunning string                    `json:"AllowOtherDownloadsWhileRunning"`
	Appid                           string                    `json:"appid"`
	AutoUpdateBehavior              string                    `json:"AutoUpdateBehavior"`
	Buildid                         string                    `json:"buildid"`
	BytesDownloaded                 string                    `json:"BytesDownloaded"`
	BytesStaged                     string                    `json:"BytesStaged"`
	BytesToDownload                 string                    `json:"BytesToDownload"`
	BytesToStage                    string                    `json:"BytesToStage"`
	DownloadType                    string                    `json:"DownloadType"`
	FullValidateAfterNextUpdate     string                    `json:"FullValidateAfterNextUpdate"`
	InstallScripts                  map[string]string         `json:"InstallScripts"`
	Installdir                      string                    `json:"installdir"`
	InstalledDepots                 map[string]InstalledDepot `json:"InstalledDepots"`
	LastOwner                       string                    `json:"LastOwner"`
	LastPlayed                      string                    `json:"LastPlayed"`
	LastUpdated                     string                    `json:"LastUpdated"`
	LauncherPath                    string                    `json:"LauncherPath"`
	MountedConfig                   BetaKey                   `json:"MountedConfig"`
	Name                            string                    `json:"name"`
	ScheduledAutoUpdate             string                    `json:"ScheduledAutoUpdate"`
	SizeOnDisk                      string                    `json:"SizeOnDisk"`
	StagingSize                     string                    `json:"StagingSize"`
	StateFlags                      string                    `json:"StateFlags"`
	TargetBuildID                   string                    `json:"TargetBuildID"`
	Universe                        string                    `json:"universe"`
	UpdateResult                    string                    `json:"UpdateResult"`
	UserConfig                      BetaKey                   `json:"UserConfig"`
}

type InstalledDepot struct {
	Manifest string `json:"manifest"`
	Size     string `json:"size"`
}

type BetaKey struct {
	BetaKey string `json:"BetaKey"`
}
