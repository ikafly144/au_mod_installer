package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func (a *App) ClearModCache() error {
	cacheDir := filepath.Join(a.ConfigDir, "mods")
	if _, err := os.Stat(cacheDir); err == nil {
		return os.RemoveAll(cacheDir)
	}
	return nil
}

func (a *App) HandleSharedProfile(uri string) (*profile.Profile, error) {
	if !strings.HasPrefix(uri, "mod-of-us://profile/") {
		return nil, fmt.Errorf("invalid URI scheme")
	}

	dataStr := strings.TrimPrefix(uri, "mod-of-us://profile/")
	data, err := base64.URLEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode profile data: %w", err)
	}

	var prof profile.Profile
	if err := json.Unmarshal(data, &prof); err != nil {
		return nil, fmt.Errorf("failed to unmarshal profile data: %w", err)
	}

	// Reset ID to avoid collision if it's a known one, but maybe better to let user decide?
	// For now, let's keep it but user should confirm import.
	return &prof, nil
}

func (a *App) ExportProfile(prof profile.Profile) (string, error) {
	data, err := json.Marshal(prof)
	if err != nil {
		return "", err
	}

	dataStr := base64.URLEncoding.EncodeToString(data)
	return "mod-of-us://profile/" + dataStr, nil
}
