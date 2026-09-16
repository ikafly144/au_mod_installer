package core

import (
	"log/slog"
	"time"

	"uuid"

	"github.com/ikafly144/modrepo/pkg/repomgr"
)

func (a *App) GetGameVersion(gamePath string) (string, error) {
	return repomgr.GetVersion(gamePath)
}

func (a *App) SetRunningPlayStartedAt(t time.Time) {
	a.runningProfileMu.Lock()
	a.runningStartedAt = t
	a.runningProfileMu.Unlock()
}

func (a *App) OnGameStartedInternal(profileID uuid.UUID, pid int) {
	a.runningProfileMu.Lock()
	wasRunning := a.runningProfileID == profileID && a.runningGamePID > 0
	a.runningProfileID = profileID
	a.runningGamePID = pid
	isRunning := a.runningProfileID == profileID && a.runningGamePID > 0
	a.runningProfileMu.Unlock()

	if wasRunning != isRunning && a.OnGameStarted != nil {
		a.OnGameStarted(profileID, pid)
	}
}

func (a *App) OnGameExitedInternal(profileID uuid.UUID) {
	a.runningProfileMu.Lock()
	wasRunning := a.runningProfileID == profileID && a.runningGamePID > 0
	a.runningProfileID = uuid.Nil()
	a.runningGamePID = 0
	a.runningStartedAt = time.Time{}
	isRunning := a.runningProfileID == profileID && a.runningGamePID > 0
	a.runningProfileMu.Unlock()

	if wasRunning != isRunning && a.OnGameExited != nil {
		a.OnGameExited(profileID)
	}
}

func (a *App) IsCurrentRunningProcess(profileID uuid.UUID, pid int) bool {
	a.runningProfileMu.Lock()
	defer a.runningProfileMu.Unlock()
	return a.runningProfileID == profileID && a.runningGamePID == pid
}

func (a *App) WatchRestoredRunningProfile(profileID uuid.UUID, pid int, startedAt time.Time, pollInterval time.Duration, onExited func()) {
	if profileID == uuid.Nil() || pid <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for range ticker.C {
			if !a.IsCurrentRunningProcess(profileID, pid) {
				return
			}
			running, err := repomgr.IsProcessRunning(pid)
			if err != nil {
				slog.Debug("Failed to check restored game process state", "profile_id", profileID, "pid", pid, "error", err)
				continue
			}
			if running {
				continue
			}
			if !a.IsCurrentRunningProcess(profileID, pid) {
				return
			}
			if onExited != nil {
				onExited()
			}
			a.OnGameExitedInternal(profileID)
			a.ClearRunningProfile(profileID)
			return
		}
	}()
}

func (a *App) CurrentRunningProfileAndPID() (uuid.UUID, int) {
	a.runningProfileMu.Lock()
	defer a.runningProfileMu.Unlock()
	return a.runningProfileID, a.runningGamePID
}

func (a *App) SetRunningProfile(profileID uuid.UUID) {
	if profileID == uuid.Nil() {
		return
	}
	a.runningProfileMu.Lock()
	a.runningProfileID = profileID
	a.runningProfileMu.Unlock()
}

func (a *App) ClearRunningProfile(profileID uuid.UUID) {
	a.runningProfileMu.Lock()
	if a.runningProfileID == profileID {
		a.runningProfileID = uuid.Nil()
	}
	a.runningProfileMu.Unlock()
}

func (a *App) SetLaunchingProfile(profileID uuid.UUID, launching bool) {
	a.runningProfileMu.Lock()
	a.launchingProfile = launching
	if launching {
		a.launchingProfileID = profileID
	} else if a.launchingProfileID == profileID {
		a.launchingProfileID = uuid.Nil()
	}
	a.runningProfileMu.Unlock()
}

func (a *App) CurrentBusyProfile() (uuid.UUID, bool) {
	a.runningProfileMu.Lock()
	defer a.runningProfileMu.Unlock()
	if a.launchingProfile {
		return a.launchingProfileID, true
	}
	return a.runningProfileID, false
}

func (a *App) IsAnyProfileBusy() bool {
	runningProfileID, launching := a.CurrentBusyProfile()
	return launching || runningProfileID != uuid.Nil()
}

func (a *App) IsProfileBusy(profileID uuid.UUID) bool {
	if profileID == uuid.Nil() {
		return false
	}
	a.runningProfileMu.Lock()
	defer a.runningProfileMu.Unlock()
	if a.launchingProfile && a.launchingProfileID == profileID {
		return true
	}
	return a.runningProfileID == profileID
}

func (a *App) IsProfileRunning(profileID uuid.UUID) bool {
	if profileID == uuid.Nil() {
		return false
	}
	a.runningProfileMu.Lock()
	defer a.runningProfileMu.Unlock()
	return a.runningProfileID == profileID && a.runningGamePID > 0
}
