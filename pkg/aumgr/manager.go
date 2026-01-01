package aumgr

import (
	"path/filepath"
	"strings"
)

type LauncherType string

const (
	LauncherUnknown   LauncherType = ""
	LauncherSteam     LauncherType = "steam"
	LauncherEpicGames LauncherType = "epic"
)

var launcherTypeNames = map[LauncherType]string{
	LauncherUnknown:   "Unknown",
	LauncherSteam:     "Steam",
	LauncherEpicGames: "Epic Games",
}

func (lt LauncherType) String() string {
	return launcherTypeNames[lt]
}

func LauncherFromString(s string) LauncherType {
	for k, v := range launcherTypeNames {
		if v == s {
			return k
		}
	}
	return LauncherUnknown
}

func DetectLauncherType(amongUsDir string) LauncherType {
	if filepath.Base(amongUsDir) == "Among Us.exe" {
		amongUsDir = filepath.Dir(amongUsDir)
	}
	if filepath.Base(amongUsDir) == "Among Us" && (strings.Contains(amongUsDir, "Steam") || strings.Contains(amongUsDir, "steamapps")) {
		return LauncherSteam
	}
	if filepath.Base(amongUsDir) == "AmongUs" && strings.Contains(amongUsDir, "Epic Games") {
		return LauncherEpicGames
	}
	return LauncherUnknown
}
