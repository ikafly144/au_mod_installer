package aumgr

import "errors"

type Manifest interface {
	GetVersion() string
}

func GetManifest(launcherType LauncherType, amongUsDir string) (Manifest, error) {
	switch launcherType {
	case LauncherSteam:
		return getSteamManifest(amongUsDir)
	case LauncherEpicGames:
		return getEpicManifest()
	default:
		return nil, errors.New("unsupported launcher type")
	}
}
