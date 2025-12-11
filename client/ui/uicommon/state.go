package uicommon

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
	"github.com/ikafly144/au_mod_installer/client/rest"
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
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
	if aumgr.DetectLauncherType(detectedPath) == aumgr.LauncherUnknown {
		return nil, errors.New("Among Us detected but launcher type is unknown")
	}

	// execPath, err := os.Executable()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get executable path: %w", err)
	// }

	// modPath := filepath.Join(filepath.Dir(execPath), "mods")

	// if err := os.MkdirAll(modPath, 0755); err != nil {
	// 	return nil, fmt.Errorf("failed to create mods directory: %w", err)
	// }

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
		Mods:             binding.BindList(&[]modmgr.Mod{}, func(a, b modmgr.Mod) bool { return a.ID == b.ID }),
		InstallSelect:    widget.NewSelect([]string{}, s.selectLauncher),
		ErrorText:        widget.NewRichTextFromMarkdown(""),

		Rest: cfg.rest,
	}

	s.ErrorText.Wrapping = fyne.TextWrapWord
	s.ErrorText.Hide()
	s.InstallSelect.PlaceHolder = lang.LocalizeKey("installer.select_install", "（Among Usを選択）")
	detectedLauncher := aumgr.DetectLauncherType(detectedPath)
	s.InstallSelect.Options = []string{detectedLauncher.String(), lang.LocalizeKey("installer.manual_select", "手動選択")}
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
	ModInstalled     binding.Bool
	CanLaunch        binding.Bool
	CanInstall       binding.Bool
	Mods             binding.ExternalList[modmgr.Mod] // Deprecated: use repository Repos instead

	Rest rest.Client

	InstallSelect *widget.Select
	ErrorText     *widget.RichText
}

func (s *State) ModInstallDir() string {
	path, err := s.SelectedGamePath.Get()
	if err != nil || path == "" {
		return ""
	}
	return path
}

func (s *State) Mod(id string) (*modmgr.Mod, error) {
	mods, err := s.Mods.Get()
	if err != nil {
		return nil, err
	}
	for _, mod := range mods {
		if mod.ID == id {
			return &mod, nil
		}
	}
	return nil, fmt.Errorf("mod not found: %s", id)
}

type Tab interface {
	Tab() (*container.TabItem, error)
}
