package core

import (
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	commonrest "github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"
)

func (a *App) GetGameVersion(gamePath string) (string, error) {
	return aumgr.GetVersion(gamePath)
}

func (a *App) IsDirectJoinEnabledForRunningProfile() bool {
	a.runningProfileMu.Lock()
	defer a.runningProfileMu.Unlock()
	return a.runningDirectJoin
}

func (a *App) SetRunningDirectJoin(enabled bool) {
	a.runningProfileMu.Lock()
	a.runningDirectJoin = enabled
	a.runningProfileMu.Unlock()
}

func (a *App) SetRunningPlayStartedAt(startedAt time.Time) {
	a.runningProfileMu.Lock()
	a.runningStartedAt = startedAt
	a.runningProfileMu.Unlock()
}

func (a *App) OnGameStartedInternal(profileID uuid.UUID, pid int) {
	a.runningProfileMu.Lock()
	wasRunning := a.runningProfileID == profileID && a.runningGamePID > 0
	a.runningGamePID = pid
	directJoin := a.runningDirectJoin
	isRunning := a.runningProfileID == profileID && a.runningGamePID > 0
	a.runningProfileMu.Unlock()

	if wasRunning != isRunning && a.OnGameStarted != nil {
		a.OnGameStarted(profileID, pid)
	}

	if !directJoin || pid <= 0 || !a.IsLobbyInfoAvailable() {
		return
	}
	a.StartLobbyPolling(pid)
}

func (a *App) OnGameExitedInternal(profileID uuid.UUID) {
	a.StopLobbyPolling()
	a.runningProfileMu.Lock()
	wasRunning := a.runningProfileID == profileID && a.runningGamePID > 0
	a.runningGamePID = 0
	a.runningDirectJoin = false
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
	if profileID == uuid.Nil || pid <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for range ticker.C {
			if !a.IsCurrentRunningProcess(profileID, pid) {
				return
			}
			running, err := aumgr.IsProcessRunning(pid)
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
			a.OnGameExitedInternal(profileID)
			a.ClearRunningProfile(profileID)
			if onExited != nil {
				onExited()
			}
			return
		}
	}()
}

func (a *App) StartLobbyPolling(pid int) {
	a.StopLobbyPolling()
	stop := a.StartLobbyInfoPolling(pid, 2*time.Second, func(info *IPCLobbyInfo) {
		a.runningProfileMu.Lock()
		a.lobbyInfo = info
		a.runningProfileMu.Unlock()
		if a.OnLobbyInfoUpdated != nil {
			a.OnLobbyInfoUpdated(info)
		}
	}, func(err error) {
		slog.Debug("Lobby polling failed", "error", err)
	})
	a.runningProfileMu.Lock()
	a.lobbyPollStop = stop
	a.runningProfileMu.Unlock()
}

func (a *App) StopLobbyPolling() {
	a.runningProfileMu.Lock()
	stop := a.lobbyPollStop
	a.lobbyPollStop = nil
	a.lobbyInfo = nil
	onLobbyInfoUpdated := a.OnLobbyInfoUpdated
	a.runningProfileMu.Unlock()
	if stop != nil {
		stop()
	}
	if onLobbyInfoUpdated != nil {
		onLobbyInfoUpdated(nil)
	}
}

func (a *App) CurrentRoomInfo(info *IPCLobbyInfo) (commonrest.RoomInfo, bool) {
	if info == nil {
		return commonrest.RoomInfo{}, false
	}
	if !info.IsConnected || (info.IsHost != nil && !*info.IsHost) {
		return commonrest.RoomInfo{}, false
	}
	if strings.TrimSpace(info.LobbyCode) == "" {
		return commonrest.RoomInfo{}, false
	}
	if info.ServerIP == "" || info.ServerPort <= 0 {
		return commonrest.RoomInfo{}, false
	}
	room := commonrest.RoomInfo{
		LobbyCode:      strings.TrimSpace(info.LobbyCode),
		ServerIP:       strings.TrimSpace(info.ServerIP),
		ServerPort:     uint16(info.ServerPort),
		MatchMakerIp:   strings.TrimSpace(info.MatchMakerIp),
		MatchMakerPort: uint16(info.MatchMakerPort),
	}
	return room, true
}

func (a *App) CurrentRunningProfileAndPID() (uuid.UUID, int) {
	a.runningProfileMu.Lock()
	defer a.runningProfileMu.Unlock()
	return a.runningProfileID, a.runningGamePID
}

func (a *App) CurrentRunningProfile() (profile.Profile, int, bool) {
	profileID, runningPID := a.CurrentRunningProfileAndPID()
	if profileID == uuid.Nil {
		return profile.Profile{}, 0, false
	}
	prof, ok := a.ProfileManager.Get(profileID)
	if !ok {
		return profile.Profile{}, 0, false
	}
	return prof, runningPID, true
}

func (a *App) SetRunningProfile(profileID uuid.UUID) {
	if profileID == uuid.Nil {
		return
	}
	a.runningProfileMu.Lock()
	a.runningProfileID = profileID
	a.runningProfileMu.Unlock()
}

func (a *App) ClearRunningProfile(profileID uuid.UUID) {
	a.runningProfileMu.Lock()
	if a.runningProfileID == profileID {
		a.runningProfileID = uuid.Nil
	}
	a.runningProfileMu.Unlock()
}

func (a *App) SetLaunchingProfile(profileID uuid.UUID, launching bool) {
	a.runningProfileMu.Lock()
	a.launchingProfile = launching
	if launching {
		a.launchingProfileID = profileID
	} else if a.launchingProfileID == profileID {
		a.launchingProfileID = uuid.Nil
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
	return launching || runningProfileID != uuid.Nil
}

func (a *App) IsProfileBusy(profileID uuid.UUID) bool {
	if profileID == uuid.Nil {
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
	if profileID == uuid.Nil {
		return false
	}
	a.runningProfileMu.Lock()
	defer a.runningProfileMu.Unlock()
	return a.runningProfileID == profileID && a.runningGamePID > 0
}

func (a *App) HasDirectJoinFeature(versions []modmgr.ModVersion) bool {
	for _, v := range versions {
		if v.HasFeature(modmgr.FeatureDirectJoin) {
			return true
		}
	}
	return false
}
