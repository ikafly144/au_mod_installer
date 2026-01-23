package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
)

type App struct {
	Version            string
	ConfigDir          string
	Rest               rest.Client
	ProfileManager     *profile.Manager
	EpicSessionManager *aumgr.EpicSessionManager
	EpicApi            *aumgr.EpicApi

	launchLock sync.Mutex
}

func New(version string, restClient rest.Client) (*App, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %w", err)
	}
	appConfigDir := filepath.Join(configDir, "au_mod_installer")
	profileManager, err := profile.NewManager(appConfigDir)
	if err != nil {
		if err := os.RemoveAll(appConfigDir); err != nil {
			return nil, fmt.Errorf("failed to remove profile path: %w", err)
		}
		profileManager, err = profile.NewManager(appConfigDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create profile manager after removal: %w", err)
		}
	}

	epicSessionManager, err := aumgr.NewEpicSessionManager(appConfigDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create epic session manager: %w", err)
	}

	return &App{
		Version:            version,
		ConfigDir:          appConfigDir,
		Rest:               restClient,
		ProfileManager:     profileManager,
		EpicSessionManager: epicSessionManager,
		EpicApi:            aumgr.NewEpicApi(),
	}, nil
}

func (a *App) DetectGamePath() (string, error) {
	return aumgr.GetAmongUsDir()
}

func (a *App) DetectLauncherType(path string) aumgr.LauncherType {
	return aumgr.DetectLauncherType(path)
}

func (a *App) IsGameRunning() (bool, error) {
	a.launchLock.Lock()
	defer a.launchLock.Unlock()

	pid, err := aumgr.IsAmongUsRunning()
	if err != nil {
		return false, err
	}
	return pid != 0, nil
}

func (a *App) UninstallMod(gamePath string, progressListener progress.Progress) error {
	modInstallLocation, err := os.OpenRoot(gamePath)
	if err != nil {
		return fmt.Errorf("failed to open game root: %w", err)
	}
	defer modInstallLocation.Close()

	if _, err := modInstallLocation.Stat(modmgr.InstallationInfoFileName); os.IsNotExist(err) {
		return fmt.Errorf("mod is not installed in this path")
	}

	return modmgr.UninstallMod(modInstallLocation, progressListener, nil)
}
