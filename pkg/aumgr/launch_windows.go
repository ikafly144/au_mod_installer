//go:build windows

package aumgr

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"golang.org/x/sys/windows"
)

func LaunchAmongUs(launcherType LauncherType, amongUsDir string, dllDir string, exchangeCode string, lobbyCode string, serverIP string, serverPort uint16, onStarted func(pid int) error) error {
	switch launcherType {
	case LauncherSteam:
		return launchSteam(amongUsDir, dllDir, lobbyCode, serverIP, serverPort, onStarted)
	case LauncherEpicGames:
		return launchEpicGames(amongUsDir, dllDir, exchangeCode, lobbyCode, serverIP, serverPort, onStarted)
	default:
		return launchDefault(amongUsDir, dllDir, lobbyCode, serverIP, serverPort, onStarted)
	}
}

func launchDefault(amongUsDir string, dllDir string, lobbyCode string, serverIP string, serverPort uint16, onStarted func(pid int) error, args ...string) error {
	exePath := filepath.Join(amongUsDir, "Among Us.exe")
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return fmt.Errorf("among Us executable not found: %s", exePath)
	}

	finalArgs := make([]string, 0, len(args)+8)
	finalArgs = append(finalArgs, args...)

	if dllDir != "" {
		slog.Info("Setting DLL directory", "dir", dllDir)
		if err := windows.SetDllDirectory(dllDir); err != nil {
			return fmt.Errorf("SetDllDirectory failed: %v", err)
		}

		// Add Doorstop arguments
		targetAssembly := filepath.Join(dllDir, "BepInEx", "core", "BepInEx.Unity.IL2CPP.dll")
		coreClrPath := filepath.Join(dllDir, "dotnet", "coreclr.dll")
		corlibDir := filepath.Join(dllDir, "dotnet")

		finalArgs = append(finalArgs,
			"--doorstop-enabled", "true",
			"--doorstop-target-assembly", targetAssembly,
			"--doorstop-clr-corlib-dir", corlibDir,
			"--doorstop-clr-runtime-coreclr-path", coreClrPath,
		)
	}
	if lobbyCode != "" {
		finalArgs = append(finalArgs, "--lobby-code", lobbyCode)
	}
	if serverIP != "" && serverPort > 0 {
		finalArgs = append(finalArgs, "--server-ip", serverIP, "--server-port", strconv.FormatUint(uint64(serverPort), 10))
	}

	cmd := exec.Command(exePath, finalArgs...)
	cmd.Dir = amongUsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Redact sensitive info in logs
	logArgs := make([]string, len(finalArgs))
	copy(logArgs, finalArgs)
	for i, arg := range logArgs {
		if len(arg) > 15 && arg[:15] == "-AUTH_PASSWORD=" {
			logArgs[i] = "-AUTH_PASSWORD=*****"
		}
	}

	slog.Info("Launching Among Us", "path", exePath, "args", logArgs)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Among Us: %w", err)
	}
	if onStarted != nil && cmd.Process != nil {
		if err := onStarted(cmd.Process.Pid); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return fmt.Errorf("launch started but failed to notify process start: %w", err)
		}
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed while running Among Us: %w", err)
	}

	return nil
}

func launchSteam(amongUsDir string, dllDir string, lobbyCode string, serverIP string, serverPort uint16, onStarted func(pid int) error) error {
	steamRunning, err := isSteamRunning()
	if err != nil {
		return fmt.Errorf("failed to check Steam process: %w", err)
	}
	if !steamRunning {
		return fmt.Errorf("cannot launch Among Us: Steam is not running. launch Steam first")
	}

	// Directly launch the executable to support SetDllDirectory
	return launchDefault(amongUsDir, dllDir, lobbyCode, serverIP, serverPort, onStarted)
}

func isSteamRunning() (bool, error) {
	processes, err := getProcesses()
	if err != nil {
		return false, err
	}
	return findProcessByName(processes, "steam.exe") != nil, nil
}

func launchEpicGames(amongUsDir string, dllDir string, exchangeCode string, lobbyCode string, serverIP string, serverPort uint16, onStarted func(pid int) error) error {
	args := []string{}
	if exchangeCode != "" {
		args = append(args, "-AUTH_PASSWORD="+exchangeCode)
		args = append(args, "-AUTH_TYPE=exchangecode")
		args = append(args, "-AUTH_LOGIN=unused")
	}
	return launchDefault(amongUsDir, dllDir, lobbyCode, serverIP, serverPort, onStarted, args...)
}
