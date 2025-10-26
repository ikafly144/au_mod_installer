package common

import (
	"au_mod_installer/pkg/aumgr"
	"au_mod_installer/pkg/modmgr"
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

func NewState(w fyne.Window, version string) (*State, error) {
	detectedPath, err := aumgr.GetAmongUsDir()
	if err != nil {
		return nil, err
	}
	if aumgr.DetectLauncherType(detectedPath) == aumgr.LauncherUnknown {
		return nil, errors.New("Among Us detected but launcher type is unknown")
	}
	var s State
	s = State{
		Version:          version,
		Window:           w,
		SelectedGamePath: binding.NewString(),
		DetectedGamePath: detectedPath,
		ModInstalled:     binding.NewBool(),
		CanLaunch:        binding.NewBool(),
		CanInstall:       binding.NewBool(),
		Mods:             binding.BindList(&[]modmgr.Mod{}, func(a, b modmgr.Mod) bool { return a.Name == b.Name && a.Version == b.Version }),
		InstallSelect:    widget.NewSelect([]string{}, s.selectLauncher),
		ErrorText:        widget.NewRichTextFromMarkdown(""),
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
	Version          string
	Window           fyne.Window
	SelectedGamePath binding.String
	DetectedGamePath string
	ModInstalled     binding.Bool
	CanLaunch        binding.Bool
	CanInstall       binding.Bool
	Mods             binding.ExternalList[modmgr.Mod]

	InstallSelect *widget.Select
	ErrorText     *widget.RichText
}

func (s *State) Mod(name string) (*modmgr.Mod, error) {
	mods, err := s.Mods.Get()
	if err != nil {
		return nil, err
	}
	for _, mod := range mods {
		if mod.Name == name {
			return &mod, nil
		}
	}
	return nil, fmt.Errorf("mod not found: %s", name)
}

type Tab interface {
	Tab() (*container.TabItem, error)
}
