//go:build windows

package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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

func (a *App) SendLobbyJoinByPID(pid int, joinInfo LaunchJoinInfo) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid: %d", pid)
	}
	pipeName := fmt.Sprintf(`\\.\pipe\LobbyUtilsPipe-%d`, pid)
	timeout := 3 * time.Second
	conn, err := winio.DialPipe(pipeName, &timeout)
	if err != nil {
		return fmt.Errorf("failed to connect lobby ipc: %w", err)
	}
	defer conn.Close()

	payload := map[string]any{
		"Action": "join",
	}
	if joinInfo.LobbyCode != "" {
		payload["Code"] = joinInfo.LobbyCode
	}
	if joinInfo.ServerIP != "" && joinInfo.ServerPort > 0 {
		payload["Ip"] = joinInfo.ServerIP
		payload["Port"] = joinInfo.ServerPort
	}
	if err := writeIPCJSON(conn, payload); err != nil {
		return err
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read lobby join response: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return fmt.Errorf("empty lobby join response")
	}
	var rs ipcResponse
	if err := json.Unmarshal([]byte(line), &rs); err != nil {
		return fmt.Errorf("failed to decode lobby join response: %w", err)
	}
	if !rs.Success {
		if rs.Message != "" {
			return fmt.Errorf("lobby join failed: %s", rs.Message)
		}
		return fmt.Errorf("lobby join failed")
	}
	return nil
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
