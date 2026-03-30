package uicommon

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2/lang"

	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/client/core"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
)

func (s *State) Launch(path string, directJoinEnabled bool) {
	s.launchLock.Lock()
	defer s.launchLock.Unlock()

	if s.Core.DetectLauncherType(path) == aumgr.LauncherEpicGames {
		if _, err := s.Core.EpicSessionManager.GetValidSession(s.Core.EpicApi); err != nil {
			s.ShowEpicLoginWindow(func() {
				go s.Launch(path, directJoinEnabled)
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
	if activeProfileID == uuid.Nil {
		s.ShowErrorDialog(errors.New(lang.LocalizeKey("launcher.error.no_profile", "Please select a profile to launch.")))
		return
	}
	profileLock, err := s.Core.AcquireProfileLaunchLock(activeProfileID)
	if err != nil {
		if errors.Is(err, core.ErrProfileLaunchBusy) {
			s.ShowErrorDialog(errors.New(lang.LocalizeKey("error.game_already_running", "Already running.")))
			return
		}
		s.ShowErrorDialog(err)
		return
	}
	defer func() {
		if err := profileLock.Release(); err != nil {
			slog.Warn("Failed to release profile launch lock", "error", err)
		}
	}()

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

	startedAt := time.Now()
	joinInfo := s.TakePendingJoinInfo()
	if err := s.Core.ExecuteLaunch(path, profileDir, joinInfo, func(pid int) error {
		if err := profileLock.SetGamePID(pid, startedAt, directJoinEnabled); err != nil {
			return err
		}
		if s.OnGameStarted != nil {
			s.OnGameStarted(activeProfileID, pid)
		}
		return nil
	}); err != nil {
		s.ShowErrorDialog(errors.New(lang.LocalizeKey("launch.error.launch_failed", "Failed to launch Among Us: ") + err.Error()))
		slog.Warn("Failed to launch Among Us", "error", err)
	}
	if s.OnGameExited != nil {
		s.OnGameExited(activeProfileID)
	}
	finishedAt := time.Now()
	if activeProfileID != uuid.Nil {
		if err := s.UpdateProfileLaunchMetrics(activeProfileID, startedAt, finishedAt); err != nil {
			s.SetError(err)
		}
	}
	_ = s.CanLaunch.Set(true)
	_ = s.CanInstall.Set(true)
}

func (s *State) UpdateProfileLaunchMetrics(profileID uuid.UUID, startedAt, finishedAt time.Time) error {
	prof, found := s.ProfileManager.Get(profileID)
	if !found {
		return nil
	}
	if startedAt.IsZero() {
		startedAt = finishedAt
	}

	if finishedAt.After(startedAt) {
		prof.AddPlayDuration(finishedAt.Sub(startedAt))
	}
	prof.LastLaunchedAt = finishedAt
	prof.UpdatedAt = finishedAt
	if err := s.ProfileManager.Add(prof); err != nil {
		return fmt.Errorf("failed to save profile launch metrics: %w", err)
	}
	return nil
}
