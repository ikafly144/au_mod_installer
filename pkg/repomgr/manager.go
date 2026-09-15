package repomgr

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	SteamAppID     = "3241660"
	ExecutableName = "REPO.exe"
	DataDirName    = "REPO_Data"
)

type LauncherType string

const (
	LauncherUnknown LauncherType = ""
	LauncherSteam   LauncherType = "steam"
	LauncherManual  LauncherType = "manual"
)

var launcherTypeNames = map[LauncherType]string{
	LauncherUnknown: "Unknown",
	LauncherSteam:   "Steam",
	LauncherManual:  "Manual Selection",
}

func (lt LauncherType) String() string {
	if name, ok := launcherTypeNames[lt]; ok {
		return name
	}
	return "Unknown"
}

func LauncherFromString(s string) LauncherType {
	for k, v := range launcherTypeNames {
		if v == s {
			return k
		}
	}
	return LauncherUnknown
}

func DetectLauncherType(gameDir string) LauncherType {
	if filepath.Base(gameDir) == ExecutableName {
		gameDir = filepath.Dir(gameDir)
	}
	if strings.Contains(strings.ToLower(gameDir), "steam") || strings.Contains(strings.ToLower(gameDir), "steamapps") {
		return LauncherSteam
	}
	if _, err := os.Stat(filepath.Join(gameDir, ExecutableName)); err == nil {
		return LauncherManual
	}
	return LauncherUnknown
}
