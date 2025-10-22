package aumgr

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func GetAmongUsDir() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, "SOFTWARE\\Classes\\amongus\\shell\\open\\command", registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()

	val, _, err := key.GetStringValue("")
	if err != nil {
		return "", err
	}
	val = strings.Trim(strings.TrimSpace(val[0:len(val)-4]), "\"")
	val, ok := strings.CutSuffix(val, "Among Us_Data\\Resources\\AmongUsHelper.exe")
	if !ok {
		return "", fmt.Errorf("Among Us Helper is not supported %s", val)
	}
	return val, nil
}

type LauncherType int

const (
	LauncherUnknown LauncherType = iota
	LauncherSteam
	LauncherEpicGames
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
	if strings.Contains(amongUsDir, "Steam") || strings.Contains(amongUsDir, "steamapps") {
		return LauncherSteam
	}
	if strings.Contains(amongUsDir, "Epic Games") {
		return LauncherEpicGames
	}
	return LauncherUnknown
}
