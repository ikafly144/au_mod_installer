package core

import (
	"os"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

type InstallStatus int

const (
	StatusNotInstalled InstallStatus = iota
	StatusInstalled
	StatusBroken
	StatusIncompatible
)

type InstallationInfo struct {
	Status               InstallStatus
	GameVersion          string
	InstalledGameVersion string
	InstalledMods        []modmgr.InstalledVersionInfo
	OutdatedMods         []OutdatedMod
	Error                error
}

type OutdatedMod struct {
	ID             string
	CurrentVersion string
	LatestVersion  string
}

// Deprecated:
func (a *App) GetInstallationStatus(gamePath string, checkUpdates bool) *InstallationInfo {
	info := &InstallationInfo{
		Status: StatusNotInstalled,
	}

	gameVersion, err := aumgr.GetVersion(gamePath)
	if err != nil {
		info.Error = err
		return info
	}
	info.GameVersion = gameVersion

	modInstallLocation, err := os.OpenRoot(gamePath)
	if err != nil {
		info.Error = err
		return info
	}
	defer modInstallLocation.Close()

	if _, err := modInstallLocation.Stat(modmgr.InstallationInfoFileName); os.IsNotExist(err) {
		return info
	}

	// TODO: remove legacy code
	//nolint:staticcheck
	installationInfo, err := modmgr.LoadInstallationInfo(modInstallLocation)
	if err != nil {
		info.Error = err
		return info
	}

	info.Status = StatusInstalled
	if installationInfo.Status == modmgr.InstallStatusBroken {
		info.Status = StatusBroken
	}

	info.InstalledGameVersion = installationInfo.InstalledGameVersion
	info.InstalledMods = installationInfo.InstalledMods

	if info.GameVersion != info.InstalledGameVersion {
		info.Status = StatusIncompatible
	}

	if checkUpdates {
		for _, mod := range installationInfo.InstalledMods {
			remoteMod, err := a.Rest.GetMod(mod.ModID)
			if err != nil {
				continue
			}
			if remoteMod.LatestVersionID != mod.VersionID {
				info.OutdatedMods = append(info.OutdatedMods, OutdatedMod{
					ID:             mod.ModID,
					CurrentVersion: mod.VersionID,
					LatestVersion:  remoteMod.LatestVersionID,
				})
			}
		}
	}

	return info
}

func (a *App) GetGameVersion(gamePath string) (string, error) {
	return aumgr.GetVersion(gamePath)
}
