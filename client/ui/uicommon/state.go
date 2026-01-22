package uicommon

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ikafly144/au_mod_installer/client/core"
	"github.com/ikafly144/au_mod_installer/client/rest"
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
	var cfg Config
	for _, option := range options {
		option(&cfg)
	}

	app, err := core.New(version, cfg.rest)
	if err != nil {
		return nil, err
	}

	detectedPath, err := app.DetectGamePath()
	if err != nil {
		return nil, err
	}

	var s State
	s = State{
		Version:          version,
		Window:           w,
		Core:             app,
		SelectedGamePath: binding.NewString(),
		DetectedGamePath: detectedPath,
		ModInstalled:     binding.NewBool(),
		CanLaunch:        binding.NewBool(),
		CanInstall:       binding.NewBool(),
		InstallSelect:    widget.NewSelect([]string{}, s.selectLauncher),
		ErrorText:        widget.NewRichTextFromMarkdown(""),

		ModInstalledInfo: widget.NewLabel(lang.LocalizeKey("installer.select_install_path", "Please select the installation path.")),
		Rest:             app.Rest,
		ProfileManager:   app.ProfileManager,
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
	detectedLauncher := app.DetectLauncherType(detectedPath)
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

	Core           *core.App
	Rest           rest.Client
	ProfileManager *profile.Manager

	ModInstalledInfo *widget.Label
	InstallSelect    *widget.Select
	ErrorText        *widget.RichText

	ActiveProfile binding.String
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

	status := i.Core.GetInstallationStatus(path, true)
	if status.Error != nil {
		slog.Warn("Failed to get installation status", "error", status.Error)
		// Assuming generic error for now, or we can check type of error
		// Using the error message from status if possible, or fallback
		i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.error.failed_to_get_version", "Mod is installed, but failed to get game version information."))
		return
	}

	// Update ModInstalled binding
	isInstalled := status.Status != core.StatusNotInstalled
	// Avoid infinite loop if binding triggers this function?
	// ModInstalled.Set triggers listener? Yes.
	// But we check i.ModInstalled.Get() in original code.
	// Here we should set it if different?
	currentInstalled, _ := i.ModInstalled.Get()
	if currentInstalled != isInstalled {
		i.ModInstalled.Set(isInstalled)
	}

	if !isInstalled {
		fyne.Do(func() {
			i.ModInstalledInfo.Refresh()
			i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.info.mod_not_installed", "Mod is not installed."))
		})
		return
	}

	defer i.ModInstalledInfo.Refresh()

	detectedLauncher := i.Core.DetectLauncherType(path)
	slog.Info("Detected launcher type", "type", detectedLauncher.String())

	if status.Status == core.StatusBroken {
		i.ModInstalledInfo.SetText(lang.LocalizeKey("installer.error.broken_installation", "Mod installation is broken. Please uninstall and reinstall the mod."))
		return
	}

	canLaunch := false
	info := lang.LocalizeKey("installer.info.mod_installed", "Mod is installed.") + "\n"

	if status.Status == core.StatusIncompatible {
		info += lang.LocalizeKey("installer.info.game_version", "Game Version: ") + status.GameVersion + " (Modインストール時: " + status.InstalledGameVersion + ")\n"
		info += lang.LocalizeKey("installer.info.mod_incompatible", "Mod is incompatible with the current game version.") + "\n"
		canLaunch = false
	} else {
		// Compatible
		info += lang.LocalizeKey("installer.info.game_version", "Game Version: ") + status.GameVersion + "\n"
		canLaunch = true

		for _, outdated := range status.OutdatedMods {
			info += lang.LocalizeKey("installer.info.mod_version_outdated", "Mod version is outdated: {{.mod}} (Installed: {{.version}}, Latest: {{.latest}})",
				map[string]any{
					"mod":     outdated.ID,
					"version": outdated.CurrentVersion,
					"latest":  outdated.LatestVersion,
				})
			canLaunch = false // TODO: allow launching with outdated mods
			// Original code broke loop here
			break
		}
	}

	var modNames []string
	for _, mod := range status.InstalledMods {
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
}

func (s *State) checkPlayingProcess() bool {
	s.launchLock.Lock()
	defer s.launchLock.Unlock()
	canInstall := false
	if ok, err := s.CanInstall.Get(); err == nil && !ok {
		canInstall = true
	}

	running, err := s.Core.IsGameRunning()
	if err != nil {
		slog.Error("Failed to check Among Us process", "error", err)
		return false
	}

	if running && !canInstall {
		// Log PID? Core doesn't return PID. That's fine.
		slog.Info("Among Us is currently running")

		_ = s.CanInstall.Set(false)
		_ = s.CanLaunch.Set(false)

		return true
	} else if canInstall && !running {
		slog.Info("Among Us is not running, re-enabling installation")
		_ = s.CanInstall.Set(true)
		fyne.Do(s.RefreshModInstallation)
	}
	return false
}
