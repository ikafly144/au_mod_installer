//go:build !windows

package core

import (
	"errors"
	"time"
)

func (a *App) GetLobbyInfoByPID(pid int) (*IPCLobbyInfo, error) {
	return nil, errors.New("lobby IPC is unsupported on this platform")
}

func (a *App) SendLobbyJoinByPID(pid int, joinInfo LaunchJoinInfo) <-chan error {
	errCh := make(chan error, 1)
	errCh <- errors.New("lobby IPC is unsupported on this platform")
	return errCh
}

func (a *App) StartLobbyInfoPolling(pid int, interval time.Duration, onInfo func(*IPCLobbyInfo), onError func(error)) func() {
	return func() {}
}

func (a *App) IsLobbyInfoAvailable() bool {
	return false
}
