//go:build windows

package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/Microsoft/go-winio"
)

type ipcResponse struct {
	Success bool         `json:"Success"`
	Action  string       `json:"Action"`
	Message string       `json:"Message"`
	Data    IPCLobbyInfo `json:"Data"`
}

func (a *App) GetLobbyInfoByPID(pid int) (*IPCLobbyInfo, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid pid: %d", pid)
	}
	pipeName := fmt.Sprintf(`\\.\pipe\LobbyUtilsPipe-%d`, pid)
	timeout := 3 * time.Second
	conn, err := winio.DialPipe(pipeName, &timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect lobby ipc: %w", err)
	}
	defer conn.Close()

	if err := writeIPCJSON(conn, map[string]any{
		"Action": "getLobbyInfo",
	}); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read lobby ipc response: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty lobby ipc response")
	}
	var rs ipcResponse
	if err := json.Unmarshal([]byte(line), &rs); err != nil {
		return nil, fmt.Errorf("failed to decode lobby ipc response: %w", err)
	}
	if !rs.Success {
		if rs.Message != "" {
			return nil, fmt.Errorf("lobby ipc failed: %s", rs.Message)
		}
		return nil, fmt.Errorf("lobby ipc failed")
	}
	return &rs.Data, nil
}

func writeIPCJSON(w io.Writer, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(raw, '\n')); err != nil {
		return err
	}
	return nil
}

const retryLobbyConnectionCount = 15

func (a *App) SendLobbyJoinByPID(pid int, joinInfo LaunchJoinInfo) <-chan error {
	errCh := make(chan error, 1)
	go a.sendLobbyJoinByPID(pid, joinInfo, retryLobbyConnectionCount, errCh)
	return errCh
}

func (a *App) sendLobbyJoinByPID(pid int, joinInfo LaunchJoinInfo, retryCount int, errCh chan error) {
	if pid <= 0 {
		errCh <- fmt.Errorf("invalid pid: %d", pid)
		return
	}
	slog.Info("Sending lobby join IPC", "pid", pid, "joinInfo", joinInfo)
	pipeName := fmt.Sprintf(`\\.\pipe\LobbyUtilsPipe-%d`, pid)
	timeout := 3 * time.Second
	conn, err := winio.DialPipe(pipeName, &timeout)
	if err != nil {
		if retryCount > 0 {
			slog.Warn("Failed to connect lobby ipc, retrying...", "error", err, "retryCount", retryCount)
			time.AfterFunc((retryLobbyConnectionCount+1-time.Duration(retryCount))*3*time.Second, func() {
				a.sendLobbyJoinByPID(pid, joinInfo, retryCount-1, errCh)
			})
			return
		}
		errCh <- fmt.Errorf("failed to connect lobby ipc: %w", err)
		return
	}
	defer conn.Close()

	payload := map[string]any{
		"Action": "join",
	}
	if joinInfo.LobbyCode != "" && joinInfo.ServerIP != "" && joinInfo.ServerPort > 0 {
		payload["Code"] = joinInfo.LobbyCode
		payload["Ip"] = joinInfo.ServerIP
		payload["Port"] = joinInfo.ServerPort
	}
	if joinInfo.MatchMakerIp != "" && joinInfo.MatchMakerPort > 0 {
		payload["MatchMakerIp"] = joinInfo.MatchMakerIp
		payload["MatchMakerPort"] = joinInfo.MatchMakerPort
	}
	if err := writeIPCJSON(conn, payload); err != nil {
		errCh <- err
		return
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		errCh <- fmt.Errorf("failed to read lobby join response: %w", err)
		return
	}
	line = strings.TrimSpace(line)
	if line == "" {
		errCh <- fmt.Errorf("empty lobby join response")
		return
	}
	var rs ipcResponse
	if err := json.Unmarshal([]byte(line), &rs); err != nil {
		errCh <- fmt.Errorf("failed to decode lobby join response: %w", err)
		return
	}
	if !rs.Success {
		if rs.Message != "" {
			errCh <- fmt.Errorf("lobby join failed: %s", rs.Message)
			return
		}
		errCh <- fmt.Errorf("lobby join failed")
		return
	}
	errCh <- nil
}

func (a *App) StartLobbyInfoPolling(pid int, interval time.Duration, onInfo func(*IPCLobbyInfo), onError func(error)) func() {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	stopCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				info, err := a.GetLobbyInfoByPID(pid)
				if err != nil {
					if onError != nil {
						onError(err)
					}
					continue
				}
				if onInfo != nil {
					onInfo(info)
				}
			}
		}
	}()
	return func() {
		close(stopCh)
	}
}

func (a *App) IsLobbyInfoAvailable() bool {
	return true
}
