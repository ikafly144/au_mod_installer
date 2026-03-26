package uicommon

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2/lang"

	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
)

func (s *State) Launch(path string) {
	s.launchLock.Lock()
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

	startedAt := time.Now()
	time.Sleep(200 * time.Millisecond)
	s.ShowGameRunningDialog()
	if err := s.Core.ExecuteLaunch(path, profileDir); err != nil {
		s.HideGameRunningDialog()
		s.ShowErrorDialog(errors.New(lang.LocalizeKey("launch.error.launch_failed", "Failed to launch Among Us: ") + err.Error()))
		slog.Warn("Failed to launch Among Us", "error", err)
		return
	}
	slog.Info("Launched Among Us successfully")
	s.waitUntilGameExited()
	s.HideGameRunningDialog()
	finishedAt := time.Now()
	if activeProfileID != uuid.Nil {
		if err := s.updateProfileLaunchMetrics(activeProfileID, startedAt, finishedAt); err != nil {
			s.SetError(err)
		}
	}
	_ = s.CanLaunch.Set(true)
	_ = s.CanInstall.Set(true)
}

func (s *State) waitUntilGameExited() {
	const (
		maxWait    = 10 * time.Second
		pollPeriod = 200 * time.Millisecond
	)
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		running, err := s.Core.IsGameRunning()
		if err != nil {
			slog.Warn("Failed to check running process after launch", "error", err)
			return
		}
		if !running {
			return
		}
		time.Sleep(pollPeriod)
	}
}

func (s *State) updateProfileLaunchMetrics(profileID uuid.UUID, startedAt, finishedAt time.Time) error {
	prof, found := s.ProfileManager.Get(profileID)
	if !found {
		return nil
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
