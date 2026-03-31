package core

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
)

type LaunchJoinInfo struct {
	LobbyCode  string
	ServerIP   string
	ServerPort uint16
}

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
		result[v.ID] = v
	}
	return slices.Collect(maps.Values(result)), nil
}

// PrepareLaunch prepares the game for launch by preparing the profile directory.
func (a *App) PrepareLaunch(gamePath string, profileID uuid.UUID) (string, func() error, error) {
	if _, err := os.Stat(filepath.Join(gamePath, "Among Us.exe")); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("among Us executable not found: %w", err)
	}

	if profileID == uuid.Nil {
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
	binaryType, err := aumgr.GetBinaryType(gamePath)
	if err != nil {
		return "", nil, err
	}

	gameVersion, err := aumgr.GetVersion(gamePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get game version: %w", err)
	}

	needSync := false

	// Check profile compatibility
	if meta, err := modmgr.GetProfileMetadata(profileDir); err == nil && meta != nil {
		if meta.GameVersion != "" && meta.GameVersion != gameVersion {
			needSync = true
		}
		if meta.BinaryType != "" && meta.BinaryType != binaryType {
			needSync = true
		}
	} else if err != nil {
		// If metadata is not found, we can assume it's an old profile and try to prepare it anyway
		return "", nil, fmt.Errorf("profile metadata not found. the profile might be created with an older version of the installer. please sync profile to update it to the latest format: %w", err)
	}

	if needSync {
		if err := a.SyncProfile(profileID, binaryType, gameVersion, nil); err != nil {
			return "", nil, fmt.Errorf("failed to sync profile: %w", err)
		}
	}

	if err := modmgr.PrepareProfileDirectory(profileDir, cacheDir, resolvedVersions, binaryType, gameVersion, false, nil); err != nil {
		return "", nil, err
	}

	cleanup := func() error {
		return nil
	}
	return profileDir, cleanup, nil
}

// SyncProfile forces a re-sync of the profile directory by clearing it and re-installing mods.
func (a *App) SyncProfile(profileID uuid.UUID, binaryType aumgr.BinaryType, gameVersion string, progressListener progress.Progress) error {
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

	return modmgr.PrepareProfileDirectory(profileDir, cacheDir, resolvedVersions, binaryType, gameVersion, true, progressListener)
}

// ExecuteLaunch launches the game and blocks until it exits.
func (a *App) ExecuteLaunch(gamePath string, dllDir string, joinInfo *LaunchJoinInfo, onStarted func(pid int) error) error {
	launcherType := aumgr.DetectLauncherType(gamePath)
	var exchangeCode string
	if launcherType == aumgr.LauncherEpicGames {
		session, err := a.EpicSessionManager.GetValidSession(a.EpicApi)
		if err == nil {
			ec, err := a.EpicApi.GetExchangeCode(session.AccessToken)
			if err == nil {
				exchangeCode = ec
			}
		}
	}
	var lobbyCode string
	var serverIP string
	var serverPort uint16
	if joinInfo != nil {
		lobbyCode = joinInfo.LobbyCode
		serverIP = joinInfo.ServerIP
		serverPort = joinInfo.ServerPort
	}
	return aumgr.LaunchAmongUs(launcherType, gamePath, dllDir, exchangeCode, lobbyCode, serverIP, serverPort, onStarted)
}
