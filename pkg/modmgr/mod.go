package modmgr

import "au_mod_installer/pkg/aumgr"

type Mod struct {
	Name     string                                 `json:"name"`
	Version  string                                 `json:"version"`
	Author   string                                 `json:"author"`
	Download map[aumgr.LauncherType]ModDownloadInfo `json:"download"`
}

type ModDownloadInfo struct {
	URL           string `json:"url"`
	TargetVersion string `json:"target_version"`
}

func (m Mod) IsCompatible(launcherType aumgr.LauncherType, gameVersion string) bool {
	if info, ok := m.Download[launcherType]; ok {
		return info.TargetVersion == gameVersion
	}
	return false
}

func (m Mod) DownloadURL(launcherType aumgr.LauncherType) string {
	if info, ok := m.Download[launcherType]; ok {
		return info.URL
	}
	return ""
}
