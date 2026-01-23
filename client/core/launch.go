package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

// PrepareLaunch prepares the game for launch by preparing the profile directory.
// It returns the path to the profile directory to be used as DLL directory.
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

	cacheDir := filepath.Join(a.ConfigDir, "mods")
	profileDir := filepath.Join(a.ConfigDir, "profiles", profileID.String())
	binaryType, err := aumgr.GetBinaryType(gamePath)
	if err != nil {
		return "", nil, err
	}

	if err := modmgr.PrepareProfileDirectory(profileDir, cacheDir, profile.Versions(), binaryType); err != nil {
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