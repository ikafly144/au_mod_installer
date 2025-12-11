//go:build windows

package aumgr

import (
	"encoding/json"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

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
