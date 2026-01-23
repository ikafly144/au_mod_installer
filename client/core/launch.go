package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
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

	result := make([]modmgr.ModVersion, 0, len(resolvedMap))
	for _, v := range resolvedMap {
		result = append(result, v)
	}
	return result, nil
}

// PrepareLaunch prepares the game for launch by preparing the profile directory.
func (a *App) PrepareLaunch(gamePath string, profileID uuid.UUID) (string, func() error, error) {
	if _, err := os.Stat(filepath.Join(gamePath, "Among Us.exe")); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("Among Us executable not found: %w", err)
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

	if err := modmgr.PrepareProfileDirectory(profileDir, cacheDir, resolvedVersions, binaryType); err != nil {
		return "", nil, err
	}

	cleanup := func() error {
		return nil
	}
	return profileDir, cleanup, nil
}

// ExecuteLaunch launches the game and blocks until it exits.
func (a *App) ExecuteLaunch(gamePath string, dllDir string) error {
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
	return aumgr.LaunchAmongUs(launcherType, gamePath, dllDir, exchangeCode)
}