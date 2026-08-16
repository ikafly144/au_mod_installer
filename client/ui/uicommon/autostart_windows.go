//go:build windows

package uicommon

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	startupRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	startupValueName    = "Mod of Us"
)

// IsAutoStartEnabled checks if the application is set to launch on Windows startup.
func IsAutoStartEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRegistryPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	val, _, err := key.GetStringValue(startupValueName)
	if err != nil {
		return false
	}
	return strings.TrimSpace(val) != ""
}

// SetAutoStartEnabled enables or disables launching the application on Windows startup.
func SetAutoStartEnabled(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, startupRegistryPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open startup registry key: %w", err)
	}
	defer key.Close()

	if enabled {
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		targetPath := execPath
		dir := filepath.Dir(execPath)
		updaterPath := filepath.Join(dir, "updater.exe")
		if _, err := os.Stat(updaterPath); err == nil {
			targetPath = updaterPath
		}
		// Register command with -silent flag so it starts in background/tray
		cmd := fmt.Sprintf("\"%s\" -silent", targetPath)
		if err := key.SetStringValue(startupValueName, cmd); err != nil {
			return fmt.Errorf("failed to write startup registry value: %w", err)
		}
		slog.Info("Enabled launch on startup in registry", "command", cmd)
		return nil
	}

	err = key.DeleteValue(startupValueName)
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("failed to delete startup registry value: %w", err)
	}
	slog.Info("Disabled launch on startup in registry")
	return nil
}

// SyncAutoStart ensures the registry startup entry matches the preference and current executable path.
func SyncAutoStart(enabled bool) {
	if enabled {
		if err := SetAutoStartEnabled(true); err != nil {
			slog.Warn("Failed to sync auto-start registry key", "error", err)
		}
	} else if IsAutoStartEnabled() {
		if err := SetAutoStartEnabled(false); err != nil {
			slog.Warn("Failed to remove auto-start registry key", "error", err)
		}
	}
}
