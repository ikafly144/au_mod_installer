package aumgr

import (
	"fmt"
	"os/exec"
)

func LaunchAmongUs(launcherType LauncherType, amongUsDir string) error {
	switch launcherType {
	case LauncherSteam:
		return launchSteam(amongUsDir)
	case LauncherEpicGames:
		return launchEpicGames(amongUsDir)
	default:
		return fmt.Errorf("unsupported launcher type: %s", launcherType)
	}
}

func launchSteam(amongUsDir string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", "steam://launch/"+steamAppID)
	return cmd.Start()
}

const (
	epicCatalogId  = "729a86a5146640a2ace9e8c595414c56"
	epicNamespace  = "33956bcb55d4452d8c47e16b94e294bd"
	epicArtifactId = "963137e4c29d4c79a81323b8fab03a40"
)

func launchEpicGames(amongUsDir string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", "com.epicgames.launcher://apps/"+epicNamespace+"%3A"+epicCatalogId+"%3A"+epicArtifactId+"?action=launch&silent=true")
	return cmd.Start()
}
