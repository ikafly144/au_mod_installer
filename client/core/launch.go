package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

// PrepareLaunch prepares the game for launch by applying mods if a profile is selected.
// It returns a cleanup function that must be called after the game exits to restore the game files.
func (a *App) PrepareLaunch(gamePath string, profileID uuid.UUID) (func() error, error) {
	if _, err := os.Stat(filepath.Join(gamePath, "Among Us.exe")); os.IsNotExist(err) {
		return nil, fmt.Errorf("Among Us executable not found: %w", err)
	}

	var restoreInfo *modmgr.RestoreInfo

	if profileID != uuid.Nil {
		profile, found := a.ProfileManager.Get(profileID)
		if found {
			configDir, err := os.UserConfigDir()
			if err != nil {
				return nil, err
			}
			cacheDir := filepath.Join(configDir, "au_mod_installer", "mods")
			binaryType, err := aumgr.GetBinaryType(gamePath)
			if err != nil {
				return nil, err
			}

			restoreInfo, err = modmgr.ApplyMods(gamePath, cacheDir, profile.Versions(), binaryType)
			if err != nil {
				return nil, err
			}
		}
	}

	cleanup := func() error {
		if restoreInfo != nil {
			return modmgr.RestoreGame(gamePath, restoreInfo)
		}
		return nil
	}
	return cleanup, nil
}

// ExecuteLaunch launches the game and blocks until it exits.
func (a *App) ExecuteLaunch(gamePath string) error {
	return aumgr.LaunchAmongUs(aumgr.DetectLauncherType(gamePath), gamePath, gamePath)
}
