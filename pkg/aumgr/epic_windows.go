//go:build windows

package aumgr

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func getEpicManifest() (Manifest, error) {
	pd, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if err != nil {
		return nil, err
	}
	manifestDirPath := filepath.Join(pd, "Epic", "EpicGamesLauncher", "Data", "Manifests")
	slog.Info("Looking for Epic Games manifests", "path", manifestDirPath)
	var amongUsManifest *EpicManifest

	if err := filepath.WalkDir(manifestDirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name()[len(d.Name())-5:] != ".item" {
			return nil
		}
		slog.Info("checking epic manifest file", "path", path)
		file, err := os.Open(path)
		if err != nil {
			slog.Error("failed to open epic manifest file", "path", path, "error", err)
			return err
		}
		defer file.Close()
		var manifest EpicManifest
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&manifest); err != nil {
			return err
		}
		if manifest.AppName == EpicArtifactId {
			amongUsManifest = &manifest
			return fs.SkipAll
		}
		return nil
	}); err != nil && !errors.Is(err, fs.SkipAll) {
		slog.Error("failed to walk epic manifest directory", "error", err)
		return nil, err
	}
	if amongUsManifest == nil {
		return nil, errors.New("among Us manifest not found in Epic Games launcher")
	}
	return amongUsManifest, nil
}
