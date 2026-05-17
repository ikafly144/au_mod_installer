package aumgr

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
)

func GetXboxAppId() (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "(Get-StartApps | Where-Object { $_.Name -like '*Among Us*' }).AppId")
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	defer stdErr.Close()
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	defer stdOut.Close()
	if err := cmd.Start(); err != nil {
		var errMsg strings.Builder
		_, ioErr := io.Copy(&errMsg, stdErr)
		if ioErr != nil {
			return "", fmt.Errorf("failed to get Xbox AppId: %w", errors.Join(err, ioErr))
		}
		return "", fmt.Errorf("failed to get Xbox AppId: %s", errMsg.String())
	}
	var appIds strings.Builder
	var ioErr error
	go func() {
		_, ioErr = io.Copy(&appIds, stdOut)
		if ioErr != nil {
			slog.Error("Failed to read Xbox AppId output", "error", ioErr)
			return
		}
	}()
	var errMsg strings.Builder
	go func() {
		_, ioErr = io.Copy(&errMsg, stdErr)
		if ioErr != nil {
			slog.Error("Failed to read Xbox AppId error output", "error", ioErr)
			return
		}
		if errMsg.Len() > 0 {
			slog.Error("Error output while getting Xbox AppId", "error", errMsg.String())
		}
	}()
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("failed to get Xbox AppId: %w", errors.Join(err, ioErr))
	}
	if cmd.ProcessState.ExitCode() != 0 {
		return "", fmt.Errorf("failed to get Xbox AppId: %s", errMsg.String())
	}

	for appId := range strings.SplitSeq(appIds.String(), "\r\n") {
		if strings.Contains(appId, "Innersloth.AmongUs") {
			return appId, nil
		}
	}
	return "", fmt.Errorf("failed to get Xbox AppId: not found")
}
