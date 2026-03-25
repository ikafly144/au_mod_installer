package uicommon

import (
	"errors"
	"log/slog"

	"fyne.io/fyne/v2/lang"

	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
)

func (s *State) Launch(path string) {
	if !s.launchLock.TryLock() {
		s.SetError(errors.New(lang.LocalizeKey("error.game_already_running", "Already running.")))
		return
	}
	defer s.launchLock.Unlock()

	if s.Core.DetectLauncherType(path) == aumgr.LauncherEpicGames {
		if _, err := s.Core.EpicSessionManager.GetValidSession(s.Core.EpicApi); err != nil {
			s.ShowEpicLoginWindow(func() {
				go s.Launch(path)
			}, nil)
			return
		}
	}

	activeProfileIDStr, _ := s.ActiveProfile.Get()
	activeProfileID, err := uuid.Parse(activeProfileIDStr)
	if err != nil {
		slog.Warn("Failed to parse active profile ID", "error", err)
		activeProfileID = uuid.Nil
	}

	profileDir, cleanup, err := s.Core.PrepareLaunch(path, activeProfileID)
	if err != nil {
		slog.Error("Failed to prepare launch", "error", err)
		s.SetError(err)
		return
	}

	defer func() {
		// Cleanup if needed (currently no-op for profile directory preservation)
		if err := cleanup(); err != nil {
			slog.Error("Failed to cleanup", "error", err)
			s.SetError(err)
		}
	}()

	if err := s.Core.ExecuteLaunch(path, profileDir); err != nil {
		s.ShowErrorDialog(errors.New(lang.LocalizeKey("launch.error.launch_failed", "Failed to launch Among Us: ") + err.Error()))
		slog.Warn("Failed to launch Among Us", "error", err)
		return
	}
	s.ShowGameRunningDialog()
	_ = s.CanLaunch.Set(true)
	_ = s.CanInstall.Set(true)
}
