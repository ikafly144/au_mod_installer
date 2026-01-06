package uicommon

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"
)

type Option func(*Config)

type Config struct {
	rest rest.Client
}

func WithRestClient(c rest.Client) func(*Config) {
	return func(cfg *Config) {
		cfg.rest = c
	}
}

func NewState(w fyne.Window, version string, options ...Option) (*State, error) {
	detectedPath, err := aumgr.GetAmongUsDir()
	if err != nil {
		return nil, err
	}

	// execPath, err := os.Executable()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get executable path: %w", err)
	// }

	// modPath := filepath.Join(filepath.Dir(execPath), "mods")

	// if err := os.MkdirAll(modPath, 0755); err != nil {
	// 	return nil, fmt.Errorf("failed to create mods directory: %w", err)
	// }

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %w", err)
	}
	profilePath := filepath.Join(configDir, "au_mod_installer")
	profileManager, err := profile.NewManager(profilePath)
	if err != nil {
		if err := os.RemoveAll(profilePath); err != nil {
			return nil, fmt.Errorf("failed to remove profile path: %w", err)
		}
		profileManager, err = profile.NewManager(profilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create profile manager after removal: %w", err)
		}
	}

	var cfg Config
	for _, option := range options {
		option(&cfg)
	}

	var s State
	s = State{
		Version: version,
		Window:  w,
		// ModPath:          modPath,
		SelectedGamePath: binding.NewString(),
		DetectedGamePath: detectedPath,
		ModInstalled:     binding.NewBool(),
		CanLaunch:        binding.NewBool(),
		CanInstall:       binding.NewBool(),
		InstallSelect:    widget.NewSelect([]string{}, s.selectLauncher),
		ErrorText:        widget.NewRichTextFromMarkdown(""),

		ModInstalledInfo: widget.NewLabel(lang.LocalizeKey("installer.select_install_path", "Please select the installation path.")),
		Rest:             cfg.rest,
		ProfileManager:   profileManager,
		ActiveProfile:    binding.NewString(),
	}

	if err := s.CanInstall.Set(true); err != nil {
		return nil, err
	}

	listener := binding.NewDataListener(s.RefreshModInstallation)
	s.ModInstalled.AddListener(listener)
	s.SelectedGamePath.AddListener(listener)
	s.ModInstalledInfo.Wrapping = fyne.TextWrapWord
	s.ModInstalledInfo.TextStyle.Symbol = true
	s.ErrorText.Wrapping = fyne.TextWrapWord
	s.ErrorText.Hide()
	s.InstallSelect.PlaceHolder = lang.LocalizeKey("installer.select_install", "(Select Among Us)")
	detectedLauncher := aumgr.DetectLauncherType(detectedPath)
	s.InstallSelect.Options = []string{detectedLauncher.String(), lang.LocalizeKey("installer.manual_select", "Manual Selection")}
	s.InstallSelect.Selected = detectedLauncher.String()
	if err := s.SelectedGamePath.Set(detectedPath); err != nil {
		return nil, err
	}

	go func() {
		time.Sleep(time.Second)
		for {
			if s.checkPlayingProcess() {
				slog.Info("Among Us is running, disabling installation and launch")
			}
			// Check every 5 seconds
			<-time.After(5 * time.Second)
		}
	}()

	return &s, nil
}

type State struct {
	Version string
	Window  fyne.Window
	// ModPath          string
	SelectedGamePath binding.String
	DetectedGamePath string
	ModInstalled     binding.Bool
	CanLaunch        binding.Bool
	CanInstall       binding.Bool
	launchLock       sync.Mutex
	installLock      sync.Mutex

	Rest rest.Client

	ModInstalledInfo *widget.Label
	InstallSelect    *widget.Select
	ErrorText        *widget.RichText

	ProfileManager *profile.Manager
	ActiveProfile  binding.String
}

func (s *State) ModInstallDir() string {
	path, err := s.SelectedGamePath.Get()
	if err != nil || path == "" {
		return ""
	}
	return path
}

type Tab interface {
	Tab() (*container.TabItem, error)
}

func (s *State) SetError(err error) {
	if err == nil {
		s.ErrorText.Hide()
		return
	}
	s.ErrorText.Segments = []widget.RichTextSegment{
		&widget.TextSegment{
			Text:  lang.LocalizeKey("common.error_occurred", "An error occurred: ") + err.Error(),
			Style: widget.RichTextStyle{ColorName: theme.ColorNameError},
		},
	}
	fyne.Do(func() {
		s.ErrorText.Refresh()
		s.ErrorText.Show()
	})
}

func (s *State) ClearError() {
	fyne.DoAndWait(s.ErrorText.Hide)
}

func (i *State) RefreshModInstallation() {
	if err := i.CanLaunch.Set(false); err != nil {
		slog.Warn("Failed to set launchable", "error", err)
	}
	path, err := i.SelectedGamePath.Get()
	if err != nil || path == "" {
		defer i.ModInstalledInfo.Refresh()
		i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.info.select_path", "Please select the installation path."))
		return
	}
	if ok, err := i.ModInstalled.Get(); ok && err == nil {
		defer i.ModInstalledInfo.Refresh()
		detectedLauncher := aumgr.DetectLauncherType(path)
		slog.Info("Detected launcher type", "type", detectedLauncher.String())
		gameVersion, err := aumgr.GetVersion(path)
		if err != nil {
			slog.Warn("Failed to get game manifest", "error", err)
			i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.error.failed_to_get_version", "Mod is installed, but failed to get game version information."))
			return
		}

		modInstallLocation, err := os.OpenRoot(i.ModInstallDir())
		if err != nil {
			slog.Warn("Failed to open game root", "error", err)
			i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.error.failed_to_open_path", "Mod is installed, but failed to open installation path."))
			return
		}

		installationInfo, err := modmgr.LoadInstallationInfo(modInstallLocation)
		if err != nil {
			slog.Warn("Failed to load installation info", "error", err)
			i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.error.failed_to_get_installation_info", "Mod is installed, but failed to get installation info."))
			return
		}
		if installationInfo.Status == modmgr.InstallStatusBroken {
			i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.error.broken_installation", "Mod installation is broken. Please uninstall and reinstall the mod."))
			return
		}
		canLaunch := false
		info := lang.LocalizeKey("installer.info.mod_installed", "Mod is installed.") + "\n"
		if gameVersion == installationInfo.InstalledGameVersion {
			info += lang.LocalizeKey("installer.info.game_version", "Game Version: ") + gameVersion + "\n"
			canLaunch = true
			for _, mod := range installationInfo.InstalledMods {
				remoteMod, err := i.Mod(mod.ModID)
				if err != nil {
					slog.Warn("Failed to get mod", "modID", mod.ID, "error", err)
					continue
				}
				if remoteMod.LatestVersion != mod.ID && remoteMod.Type != modmgr.ModTypeLibrary {
					info += lang.LocalizeKey("installer.info.mod_version_outdated", "Mod version is outdated: {{.mod}} (Installed: {{.version}}, Latest: {{.latest}})",
						map[string]any{
							"mod":     mod.ModID,
							"version": mod.ID,
							"latest":  remoteMod.LatestVersion,
						}) + "\n"
					canLaunch = false // TODO: allow launching with outdated mods
					break
				}
			}
		} else {
			info += lang.LocalizeKey("installer.info.game_version", "Game Version: ") + gameVersion + " (Modインストール時: " + installationInfo.InstalledGameVersion + ")\n"
			info += lang.LocalizeKey("installer.info.mod_incompatible", "Mod is incompatible with the current game version.") + "\n"
			installationInfo.Status = modmgr.InstallStatusIncompatible
			if err := modmgr.SaveInstallationInfo(modInstallLocation, installationInfo); err != nil {
				slog.Warn("Failed to save installation info", "error", err)
			}
			canLaunch = false
		}
		var modNames []string
		for _, mod := range installationInfo.InstalledMods {
			modNames = append(modNames, mod.ModID+" ("+mod.ID+")")
		}
		info += lang.LocalizeKey("installer.info.mod_name", "Mod: ") + strings.Join(modNames, ", ") + "\n"
		i.ModInstalledInfo.SetText(strings.TrimSpace(info))
		if strings.Contains(i.Version, "(devel)") { // NOTE: allow launching in development mode
			canLaunch = true
		}
		if err := i.CanLaunch.Set(canLaunch); err != nil {
			slog.Warn("Failed to set launchable", "error", err)
		}
	} else if err == nil {
		fyne.Do(func() {
			i.ModInstalledInfo.Refresh()
			i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.info.mod_not_installed", "Mod is not installed."))
		})
	} else {
		slog.Warn("Failed to get mod installed", "error", err)
	}
}

func (s *State) checkPlayingProcess() bool {
	s.launchLock.Lock()
	defer s.launchLock.Unlock()
	canInstall := false
	if ok, err := s.CanInstall.Get(); err == nil && !ok {
		canInstall = true
	}
	pid, err := aumgr.IsAmongUsRunning()
	if err != nil {
		slog.Error("Failed to check Among Us process", "error", err)
		return false
	}
	if pid != 0 && !canInstall {
		slog.Info("Among Us is currently running", "pid", pid)

		_ = s.CanInstall.Set(false)
		_ = s.CanLaunch.Set(false)

		return true
	} else if canInstall && pid == 0 {
		slog.Info("Among Us is not running, re-enabling installation")
		_ = s.CanInstall.Set(true)
		fyne.Do(s.RefreshModInstallation)
	}
	return false
}
