package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
)

var ErrProfileLaunchBusy = errors.New("profile launch is already running")

const (
	profileLockStateStarting = "starting"
	profileLockStateRunning  = "running"
)

type profileLaunchLockState struct {
	State             string `json:"state"`
	StarterPID        int    `json:"starter_pid,omitempty"`
	GamePID           int    `json:"game_pid,omitempty"`
	DirectJoinEnabled bool   `json:"direct_join_enabled,omitempty"`
	StartedAtUnixNano int64  `json:"started_at_unix_nano,omitempty"`
}

type ProfileLaunchLock struct {
	path      string
	profileID uuid.UUID
	released  bool
	mu        sync.Mutex
}

func (a *App) AcquireProfileLaunchLock(profileID uuid.UUID) (*ProfileLaunchLock, error) {
	if profileID == uuid.Nil {
		return &ProfileLaunchLock{}, nil
	}

	lockDir := filepath.Join(a.ConfigDir, "profile_locks")
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create profile lock directory: %w", err)
	}

	lockPath := filepath.Join(lockDir, profileID.String()+".lock")
	for range 2 {
		if err := writeInitialProfileLock(lockPath); err == nil {
			return &ProfileLaunchLock{path: lockPath, profileID: profileID}, nil
		} else if !errors.Is(err, fs.ErrExist) {
			return nil, fmt.Errorf("failed to acquire profile lock: %w", err)
		}
		busy, err := reconcileProfileLock(lockPath)
		if err != nil {
			return nil, err
		}
		if busy {
			return nil, fmt.Errorf("%w: %s", ErrProfileLaunchBusy, profileID.String())
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrProfileLaunchBusy, profileID.String())
}

func (l *ProfileLaunchLock) SetGamePID(gamePID int, startedAt time.Time, directJoinEnabled bool) error {
	if l == nil || l.path == "" || l.profileID == uuid.Nil {
		return nil
	}
	if gamePID <= 0 {
		return fmt.Errorf("invalid game process id: %d", gamePID)
	}
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return fmt.Errorf("profile lock already released")
	}
	state := profileLaunchLockState{
		State:             profileLockStateRunning,
		StarterPID:        os.Getpid(),
		GamePID:           gamePID,
		DirectJoinEnabled: directJoinEnabled,
		StartedAtUnixNano: startedAt.UnixNano(),
	}
	if err := writeProfileLockState(l.path, state); err != nil {
		return fmt.Errorf("failed to update profile lock with game pid: %w", err)
	}
	return nil
}

func (l *ProfileLaunchLock) Release() error {
	if l == nil || l.path == "" || l.profileID == uuid.Nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return nil
	}
	l.released = true
	if err := os.Remove(l.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to release profile lock: %w", err)
	}
	return nil
}

func writeInitialProfileLock(lockPath string) error {
	file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	state := profileLaunchLockState{
		State:      profileLockStateStarting,
		StarterPID: os.Getpid(),
	}
	if err := json.NewEncoder(file).Encode(state); err != nil {
		_ = os.Remove(lockPath)
		return fmt.Errorf("failed to write initial profile lock state: %w", err)
	}
	return nil
}

func writeProfileLockState(lockPath string, state profileLaunchLockState) error {
	file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(state)
}

func readProfileLockState(lockPath string) (profileLaunchLockState, error) {
	file, err := os.Open(lockPath)
	if err != nil {
		return profileLaunchLockState{}, err
	}
	defer file.Close()
	var state profileLaunchLockState
	if err := json.NewDecoder(file).Decode(&state); err != nil {
		return profileLaunchLockState{}, err
	}
	return state, nil
}

func reconcileProfileLock(lockPath string) (bool, error) {
	state, err := readProfileLockState(lockPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		if removeErr := os.Remove(lockPath); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			return true, fmt.Errorf("failed to remove invalid profile lock: %w", removeErr)
		}
		return false, nil
	}

	switch state.State {
	case profileLockStateStarting:
		if state.StarterPID <= 0 {
			if err := os.Remove(lockPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return true, fmt.Errorf("failed to remove malformed starting profile lock: %w", err)
			}
			return false, nil
		}
		running, err := aumgr.IsProcessRunning(state.StarterPID)
		if err != nil {
			return true, fmt.Errorf("failed to check starter process for profile lock: %w", err)
		}
		if running {
			return true, nil
		}
		if err := os.Remove(lockPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return true, fmt.Errorf("failed to remove stale starting profile lock: %w", err)
		}
		return false, nil
	case profileLockStateRunning:
		if state.GamePID <= 0 {
			if err := os.Remove(lockPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return true, fmt.Errorf("failed to remove malformed running profile lock: %w", err)
			}
			return false, nil
		}
		running, err := aumgr.IsProcessRunning(state.GamePID)
		if err != nil {
			return true, fmt.Errorf("failed to check game process for profile lock: %w", err)
		}
		if running {
			return true, nil
		}
		if err := os.Remove(lockPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return true, fmt.Errorf("failed to remove stale running profile lock: %w", err)
		}
		return false, nil
	default:
		if err := os.Remove(lockPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return true, fmt.Errorf("failed to remove unknown profile lock state: %w", err)
		}
		return false, nil
	}
}

type RunningProfileInfo struct {
	ProfileID         uuid.UUID
	GamePID           int
	DirectJoinEnabled bool
	PlayStartedAt     time.Time
}

// LoadRunningProfilesFromLocks scans the profile lock directory and returns information
// about profiles that are currently running (have valid lock files with active processes).
func (a *App) LoadRunningProfilesFromLocks() ([]RunningProfileInfo, error) {
	lockDir := filepath.Join(a.ConfigDir, "profile_locks")
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create profile lock directory: %w", err)
	}

	entries, err := os.ReadDir(lockDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile lock directory: %w", err)
	}

	var runningProfiles []RunningProfileInfo
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".lock" {
			continue
		}

		lockPath := filepath.Join(lockDir, entry.Name())
		busy, err := reconcileProfileLock(lockPath)
		if err != nil {
			continue
		}
		if !busy {
			continue
		}

		state, err := readProfileLockState(lockPath)
		if err != nil {
			continue
		}

		profileIDStr := strings.TrimSuffix(entry.Name(), ".lock")
		profileID, err := uuid.Parse(profileIDStr)
		if err != nil {
			continue
		}

		if state.State == profileLockStateRunning && state.GamePID > 0 {
			playStartedAt := time.Time{}
			if state.StartedAtUnixNano > 0 {
				playStartedAt = time.Unix(0, state.StartedAtUnixNano)
			}
			runningProfiles = append(runningProfiles, RunningProfileInfo{
				ProfileID:         profileID,
				GamePID:           state.GamePID,
				DirectJoinEnabled: state.DirectJoinEnabled,
				PlayStartedAt:     playStartedAt,
			})
		}
	}

	return runningProfiles, nil
}
