package core

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"uuid"

	"github.com/ikafly144/modrepo/pkg/modmgr"
	"github.com/ikafly144/modrepo/pkg/progress"
	"github.com/ikafly144/modrepo/pkg/repomgr"
)

// ResolveProfileDependencies resolves all required dependencies for the given profile.
func (a *App) ResolveProfileDependencies(profileID uuid.UUID) ([]modmgr.ModVersion, error) {
	profile, found := a.ProfileManager.Get(profileID)
	if !found {
		return nil, fmt.Errorf("profile not found: %s", profileID)
	}
	return a.ResolveDependencies(profile.Versions())
}

func (a *App) ResolveDependencies(initialMods []modmgr.ModVersion) ([]modmgr.ModVersion, error) {
	resolvedMap, err := modmgr.ResolveDependencies(initialMods, a.Rest)
	if err != nil {
		return nil, err
	}

	result := make(map[string]modmgr.ModVersion, len(resolvedMap))
	for _, v := range resolvedMap {
		result[v.ModID+"@"+v.VersionID] = v
	}
	return slices.Collect(maps.Values(result)), nil
}

// PrepareLaunch prepares the game for launch by preparing the profile directory.
func (a *App) PrepareLaunch(gamePath string, profileID uuid.UUID) (string, func() error, error) {
	if _, err := os.Stat(filepath.Join(gamePath, repomgr.ExecutableName)); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("R.E.P.O. executable not found: %w", err)
	}

	if profileID == uuid.Nil() {
		return "", func() error { return nil }, nil
	}

	profile, found := a.ProfileManager.Get(profileID)
	if !found {
		return "", nil, fmt.Errorf("profile not found: %s", profileID)
	}

	resolvedVersions, err := a.ResolveDependencies(profile.Versions())
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	cacheDir := filepath.Join(a.ConfigDir, "mods")
	profileDir := filepath.Join(a.ConfigDir, "profiles", profileID.String())
	binaryType := repomgr.BinaryType64Bit

	gameVersion, err := repomgr.GetVersion(gamePath)
	if err != nil {
		gameVersion = "unknown"
	}

	needSync := false

	// Check profile compatibility
	if meta, err := modmgr.GetProfileMetadata(profileDir); err == nil && meta != nil {
		if meta.GameVersion != "" && meta.GameVersion != gameVersion {
			needSync = true
		}
	} else if err != nil {
		return "", nil, fmt.Errorf("profile metadata error: %w", err)
	}

	if needSync {
		if err := a.SyncProfile(profileID, binaryType, gameVersion, nil); err != nil {
			return "", nil, fmt.Errorf("failed to sync profile: %w", err)
		}
	}

	// Ensure mods are downloaded to cache
	if err := modmgr.DownloadMods(cacheDir, resolvedVersions, binaryType, nil, false); err != nil {
		return "", nil, fmt.Errorf("failed to download mods: %w", err)
	}

	if err := modmgr.PrepareProfileDirectory(profileDir, gamePath, cacheDir, resolvedVersions, binaryType, gameVersion, false, nil); err != nil {
		return "", nil, err
	}

	cleanup := func() error {
		return nil
	}
	return profileDir, cleanup, nil
}

// SyncProfile forces a re-sync of the profile directory by clearing it and re-installing mods.
func (a *App) SyncProfile(profileID uuid.UUID, binaryType repomgr.BinaryType, gameVersion string, progressListener progress.Progress) error {
	profile, found := a.ProfileManager.Get(profileID)
	if !found {
		return fmt.Errorf("profile not found: %s", profileID)
	}

	resolvedVersions, err := a.ResolveDependencies(profile.Versions())
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	cacheDir := filepath.Join(a.ConfigDir, "mods")
	profileDir := filepath.Join(a.ConfigDir, "profiles", profileID.String())

	if err := modmgr.DownloadMods(cacheDir, resolvedVersions, binaryType, progressListener, true); err != nil {
		return fmt.Errorf("failed to download mods: %w", err)
	}

	return modmgr.PrepareProfileDirectory(profileDir, "", cacheDir, resolvedVersions, binaryType, gameVersion, true, progressListener)
}

// ExecuteLaunch launches R.E.P.O. and blocks until it exits.
func (a *App) ExecuteLaunch(gamePath string, dllDir string, onStarted func(pid int) error) error {
	launcherType := repomgr.DetectLauncherType(gamePath)
	return repomgr.Launch(launcherType, gamePath, dllDir, onStarted)
}
