package uicommon

import (
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
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
		ActiveProfile:    binding.BindPreferenceString("core.active_profile", fyne.CurrentApp().Preferences()),
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
	dialogLock       sync.Mutex
	activeDialog     dialog.Dialog
	runningDialog    dialog.Dialog

	Core           *core.App
	Rest           rest.Client
	ProfileManager *profile.Manager

	ModInstalledInfo *widget.Label
	InstallSelect    *widget.Select
	ErrorText        *widget.RichText

	ActiveProfile binding.String
	SharedURI     string

	OnSharedURIReceived func(uri string)
	OnDroppedURIs       func([]fyne.URI)
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
		s.ClearError()
		return
	}
	s.ShowErrorDialog(errors.New(lang.LocalizeKey("common.error_occurred", "An error occurred: ") + err.Error()))
}

func (s *State) ShowErrorDialog(err error) {
	if err == nil || s.Window == nil {
		return
	}
	s.showDialog(func() dialog.Dialog {
		return dialog.NewError(err, s.Window)
	})
}

func (s *State) ShowInfoDialog(title, message string) {
	if title == "" || message == "" || s.Window == nil {
		return
	}
	s.showDialog(func() dialog.Dialog {
		return dialog.NewInformation(title, message, s.Window)
	})
}

func (s *State) ShowGameRunningDialog() {
	if s.Window == nil {
		return
	}
	fyne.Do(func() {
		s.dialogLock.Lock()
		if s.runningDialog != nil {
			s.dialogLock.Unlock()
			return
		}
		label := widget.NewLabel(lang.LocalizeKey("launch.running.dialog", "Among Us is running. This dialog closes automatically when the game exits."))
		label.Wrapping = fyne.TextWrapWord
		d := dialog.NewCustomWithoutButtons(
			lang.LocalizeKey("launch.running.title", "Game Running"),
			container.NewVBox(label),
			s.Window,
		)
		d.SetOnClosed(func() {
			s.dialogLock.Lock()
			if s.runningDialog == d {
				s.runningDialog = nil
			}
			s.dialogLock.Unlock()
		})
		d.Resize(fyne.NewSize(420, 120))
		s.runningDialog = d
		s.dialogLock.Unlock()
		d.Show()
	})
}

func (s *State) HideGameRunningDialog() {
	fyne.Do(func() {
		s.dialogLock.Lock()
		d := s.runningDialog
		s.runningDialog = nil
		s.dialogLock.Unlock()
		if d != nil {
			d.Hide()
		}
	})
}

func (s *State) showDialog(factory func() dialog.Dialog) {
	fyne.Do(func() {
		s.dialogLock.Lock()
		prev := s.activeDialog
		s.activeDialog = nil
		s.dialogLock.Unlock()

		if prev != nil {
			prev.Hide()
		}

		d := factory()
		d.SetOnClosed(func() {
			s.dialogLock.Lock()
			if s.activeDialog == d {
				s.activeDialog = nil
			}
			s.dialogLock.Unlock()
		})

		s.dialogLock.Lock()
		s.activeDialog = d
		s.dialogLock.Unlock()
		d.Show()
	})
}

func (s *State) ClearError() {
	fyne.DoAndWait(func() {
		s.ErrorText.Hide()

		s.dialogLock.Lock()
		d := s.activeDialog
		s.activeDialog = nil
		s.dialogLock.Unlock()
		if d != nil {
			d.Hide()
		}
	})
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
		if err := i.ModInstalled.Set(isInstalled); err != nil {
			slog.Warn("Failed to set modInstalled", "error", err)
		}
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
	var info strings.Builder
	info.WriteString(lang.LocalizeKey("installer.info.mod_installed", "Mod is installed.") + "\n")

	if status.Status == core.StatusIncompatible {
		info.WriteString(lang.LocalizeKey("installer.info.game_version", "Game Version: ") + status.GameVersion + " (Modインストール時: " + status.InstalledGameVersion + ")\n")
		info.WriteString(lang.LocalizeKey("installer.info.mod_incompatible", "Mod is incompatible with the current game version.") + "\n")
		canLaunch = false
	} else {
		// Compatible
		info.WriteString(lang.LocalizeKey("installer.info.game_version", "Game Version: ") + status.GameVersion + "\n")
		canLaunch = true

		for _, outdated := range status.OutdatedMods {
			info.WriteString(lang.LocalizeKey("installer.info.mod_version_outdated", "Mod version is outdated: {{.mod}} (Installed: {{.version}}, Latest: {{.latest}})",
				map[string]any{
					"mod":     outdated.ID,
					"version": outdated.CurrentVersion,
					"latest":  outdated.LatestVersion,
				}))
			// Original code broke loop here
			break
		}
	}

	var modNames []string
	for _, mod := range status.InstalledMods {
		modNames = append(modNames, mod.ModID+" ("+mod.ID+")")
	}
	info.WriteString(lang.LocalizeKey("installer.info.mod_name", "Mod: ") + strings.Join(modNames, ", ") + "\n")
	i.ModInstalledInfo.SetText(strings.TrimSpace(info.String()))

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
	installDisabled := false
	if ok, err := s.CanInstall.Get(); err == nil && !ok {
		installDisabled = true
	}

	running, err := s.Core.IsGameRunning()
	if err != nil {
		slog.Error("Failed to check Among Us process", "error", err)
		return false
	}

	if running {
		_ = s.CanInstall.Set(false)
		_ = s.CanLaunch.Set(false)
		s.ShowGameRunningDialog()
		return true
	}

	s.HideGameRunningDialog()

	if installDisabled {
		slog.Info("Among Us is not running, re-enabling installation")
		_ = s.CanInstall.Set(true)
		fyne.Do(s.RefreshModInstallation)
	}
	return false
}
