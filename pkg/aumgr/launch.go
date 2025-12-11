package aumgr

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func LaunchAmongUs(launcherType LauncherType, amongUsDir string, dllDir string, args ...string) error {
	switch launcherType {
	case LauncherEpicGames:
		return launchEpicGames(amongUsDir, dllDir, args...)
	default:
		return launchDefault(amongUsDir, dllDir, args...)
	}
}

func launchDefault(amongUsDir string, dllDir string, args ...string) error {
	cmd := exec.Command(filepath.Join(amongUsDir, "Among Us.exe"))
	// if dllDir != "" {
	// 	if err := windows.SetDllDirectory(dllDir); err != nil {
	// 		return fmt.Errorf("SetDllDirectory failed: %v", err)
	// 	}
	// }

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Among Us: %w", err)
	}

	return nil
}

const (
	epicCatalogId  = "729a86a5146640a2ace9e8c595414c56"
	epicNamespace  = "33956bcb55d4452d8c47e16b94e294bd"
	epicArtifactId = "963137e4c29d4c79a81323b8fab03a40"
)

// TODO: implement authentication with Epic Games Launcher
func launchEpicGames(amongUsDir string, dllDir string, args ...string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", "com.epicgames.launcher://apps/"+epicNamespace+"%3A"+epicCatalogId+"%3A"+epicArtifactId+"?action=launch&silent=true")
	return cmd.Start()
}
