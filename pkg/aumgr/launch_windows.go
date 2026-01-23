//go:build windows

package aumgr

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func LaunchAmongUs(launcherType LauncherType, amongUsDir string, dllDir string, exchangeCode string) error {
	switch launcherType {
	case LauncherSteam:
		return launchSteam(amongUsDir, dllDir)
	case LauncherEpicGames:
		return launchEpicGames(amongUsDir, dllDir, exchangeCode)
	default:
		return launchDefault(amongUsDir, dllDir)
	}
}

func launchDefault(amongUsDir string, dllDir string, args ...string) error {
	exePath := filepath.Join(amongUsDir, "Among Us.exe")
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return fmt.Errorf("Among Us executable not found: %s", exePath)
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

	cmd := exec.Command(exePath, finalArgs...)
	cmd.Dir = amongUsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	slog.Info("Launching Among Us", "path", exePath, "args", finalArgs)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start Among Us: %w", err)
	}

	return nil
}

func launchSteam(amongUsDir string, dllDir string) error {
	// Directly launch the executable to support SetDllDirectory
	return launchDefault(amongUsDir, dllDir)
}

func launchEpicGames(amongUsDir string, dllDir string, exchangeCode string) error {
	args := []string{}
	if exchangeCode != "" {
		args = append(args, "-AUTH_PASSWORD="+exchangeCode)
		args = append(args, "-AUTH_TYPE=exchangecode")
		args = append(args, "-AUTH_LOGIN=unused")
	}
	return launchDefault(amongUsDir, dllDir, args...)
}