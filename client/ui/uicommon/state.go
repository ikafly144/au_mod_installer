package uicommon

import (
	"errors"
	"log/slog"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"

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
		// TODO: インストールが正常に選択されていない状態でバグらないことを検証する
		slog.Warn("Failed to detect game path", "error", err)
		detectedPath = ""
	}

	var s State
	s = State{
		Version:          version,
		Window:           w,
		Core:             app,
		SelectedGamePath: binding.NewString(),
		DetectedGamePath: detectedPath,
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

	return &s, nil
}

type State struct {
	Version string
	Window  fyne.Window
	// ModPath          string
	SelectedGamePath binding.String
	DetectedGamePath string
	CanLaunch        binding.Bool
	CanInstall       binding.Bool
	launchLock       sync.Mutex
	joinInfoLock     sync.Mutex
	dialogLock       sync.Mutex
	activeDialog     dialog.Dialog

	Core           *core.App
	Rest           rest.Client
	ProfileManager *profile.Manager

	ModInstalledInfo *widget.Label
	InstallSelect    *widget.Select
	ErrorText        *widget.RichText

	ActiveProfile binding.String
	SharedURI     string
	SharedArchive string

	OnSharedURIReceived     func(uri string)
	OnSharedArchiveReceived func(path string)
	OnDroppedURIs           func([]fyne.URI)
	OnGameStarted           func(profileID uuid.UUID, pid int)
	OnGameExited            func(profileID uuid.UUID)
	OnProfileMetricsUpdated func(profileID uuid.UUID)

	pendingJoinInfo *core.LaunchJoinInfo
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
	fyne.Do(func() {
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

func (s *State) SetPendingJoinInfo(joinInfo *core.LaunchJoinInfo) {
	s.joinInfoLock.Lock()
	defer s.joinInfoLock.Unlock()
	if joinInfo == nil {
		s.pendingJoinInfo = nil
		return
	}
	cp := *joinInfo
	s.pendingJoinInfo = &cp
}

func (s *State) TakePendingJoinInfo() *core.LaunchJoinInfo {
	s.joinInfoLock.Lock()
	defer s.joinInfoLock.Unlock()
	if s.pendingJoinInfo == nil {
		return nil
	}
	cp := *s.pendingJoinInfo
	s.pendingJoinInfo = nil
	return &cp
}
