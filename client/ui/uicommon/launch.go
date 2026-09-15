package uicommon

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2/lang"

	"uuid"

	"github.com/ikafly144/modrepo/client/core"
)

func (s *State) Launch(path string) {
	s.launchLock.Lock()
	defer s.launchLock.Unlock()

	activeProfileIDStr, _ := s.ActiveProfile.Get()
	activeProfileID, err := uuid.Parse(activeProfileIDStr)
	if err != nil {
		slog.Warn("Failed to parse active profile ID", "error", err)
		activeProfileID = uuid.Nil()
	}
	if activeProfileID == uuid.Nil() {
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
		if err := cleanup(); err != nil {
			slog.Error("Failed to cleanup", "error", err)
			s.SetError(err)
		}
	}()

	startedAt := time.Now()
	var launchSucceeded bool
	if err := s.Core.ExecuteLaunch(path, profileDir, func(pid int) error {
		if err := profileLock.SetGamePID(pid, startedAt, false); err != nil {
			return err
		}
		s.Core.SetRunningPlayStartedAt(startedAt)
		s.Core.OnGameStartedInternal(activeProfileID, pid)
		launchSucceeded = true

		go func() {
			time.Sleep(1 * time.Second)
			Alert(
				lang.LocalizeKey("notification.game_launched.title", "Game Launched"),
				lang.LocalizeKey("notification.game_launched.message", "R.E.P.O. has been launched."),
			)
		}()
		return nil
	}); err != nil {
		slog.Error("Failed to execute launch", "error", err)
		Alert(
			lang.LocalizeKey("notification.game_launch_failed.title", "Launch Failed"),
			lang.LocalizeKey("notification.game_launch_failed.message", fmt.Sprintf("Failed to launch game: %s", err.Error())),
		)
		s.SetError(err)
		return
	}

	if !launchSucceeded {
		return
	}

	playDuration := time.Since(startedAt)
	if prof, ok := s.ProfileManager.Get(activeProfileID); ok {
		prof.AddPlayDuration(playDuration)
		prof.LastLaunchedAt = time.Now()
		if err := s.ProfileManager.Update(prof); err != nil {
			slog.Warn("Failed to update profile play duration", "error", err)
		}
		if s.OnProfileMetricsUpdated != nil {
			s.OnProfileMetricsUpdated(activeProfileID)
		}
	}
	s.Core.OnGameExitedInternal(activeProfileID)
}
