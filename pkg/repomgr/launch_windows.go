//go:build windows

package repomgr

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
)

func LaunchRepo(gameDir string, dllDir string, onStarted func(pid int) error, args ...string) error {
	exePath := filepath.Join(gameDir, ExecutableName)
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return fmt.Errorf("R.E.P.O. executable not found: %s", exePath)
	}

	// Ensure Steam is running
	if err := ensureSteamRunning(); err != nil {
		slog.Warn("Could not ensure Steam is running", "error", err)
	}

	finalArgs := make([]string, 0, len(args)+4)
	finalArgs = append(finalArgs, args...)

	if dllDir != "" {
		slog.Info("Setting DLL directory for Doorstop", "dir", dllDir)
		if err := windows.SetDllDirectory(dllDir); err != nil {
			return fmt.Errorf("SetDllDirectory failed: %v", err)
		}
		defer func() {
			_ = windows.SetDllDirectory("")
		}()

		// Unity Mono Doorstop target assembly
		targetAssembly := filepath.Join(dllDir, "BepInEx", "core", "BepInEx.Preloader.dll")

		finalArgs = append(finalArgs,
			"--doorstop-enabled", "true",
			"--doorstop-target-assembly", targetAssembly,
		)
	}

	cmd := exec.Command(exePath, finalArgs...)
	cmd.Dir = gameDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	slog.Info("Launching R.E.P.O.", "path", exePath, "args", finalArgs)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start R.E.P.O.: %w", err)
	}

	if onStarted != nil && cmd.Process != nil {
		if err := onStarted(cmd.Process.Pid); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return fmt.Errorf("launch started but failed to notify process start: %w", err)
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed while running R.E.P.O.: %w", err)
	}

	return nil
}

func ensureSteamRunning() error {
	processes, err := getProcesses()
	if err != nil {
		return fmt.Errorf("failed to get process list: %w", err)
	}

	_, found := findProcessByName(processes, "steam.exe")
	if found {
		return nil
	}
	_, found = findProcessByName(processes, "steamservice.exe")
	if found {
		return nil
	}

	// Try starting Steam via steam protocol
	cmd := exec.Command("explorer.exe", "steam://")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch Steam: %w", err)
	}

	timer := time.NewTimer(15 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer timer.Stop()
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			return nil // proceed anyway
		case <-ticker.C:
			pList, err := getProcesses()
			if err != nil {
				continue
			}
			if _, found := findProcessByName(pList, "steam.exe"); found {
				return nil
			}
		}
	}
}
