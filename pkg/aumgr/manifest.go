package aumgr

type Manifest interface {
	// Deprecated: use GetVersion instead
	GetVersion() string
}

func GetManifest(launcherType LauncherType, amongUsDir string) (Manifest, error) {
	switch launcherType {
	case LauncherSteam:
		return getSteamManifest(amongUsDir)
	case LauncherEpicGames:
		return getEpicManifest()
	default:
		return UnknownManifest{}, nil
	}
}

type UnknownManifest struct{}

func (m UnknownManifest) GetVersion() string {
	return "unknown"
}
