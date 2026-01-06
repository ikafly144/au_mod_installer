package settings

import (
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/google/uuid"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/ikafly144/au_mod_installer/common/versioning"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/profile"
)

type Settings struct {
	state               *uicommon.State
	BranchSelect        *widget.Select
	ImportProfileButton *widget.Button
}

func NewSettings(state *uicommon.State) *Settings {
	branchOptions := []string{
		versioning.BranchStable.String(),
		versioning.BranchPreview.String(),
		versioning.BranchBeta.String(),
		versioning.BranchCanary.String(),
		versioning.BranchDev.String(),
	}
	branchSelect := widget.NewSelect(branchOptions, nil)
	branchSelect.PlaceHolder = lang.LocalizeKey("settings.select_update_channel", "Select Update Channel")
	branchSelect.OnChanged = func(s string) {
		fyne.CurrentApp().Preferences().SetString("core.update_branch", s)
	}
	currentBranch := fyne.CurrentApp().Preferences().StringWithFallback("core.update_branch", "stable")
	branchSelect.SetSelected(currentBranch)

	s := &Settings{
		state:        state,
		BranchSelect: branchSelect,
	}

	s.ImportProfileButton = widget.NewButtonWithIcon(lang.LocalizeKey("settings.import_profile", "Import Profile from Current Installation"), theme.DocumentSaveIcon(), s.importProfile)

	return s
}

func (s *Settings) Tab() (*container.TabItem, error) {
	list := container.NewVScroll(container.NewVBox(
		settingsEntry(lang.LocalizeKey("settings.update_channel", "Update Channel"), s.BranchSelect),
		widget.NewSeparator(),
		settingsEntry(lang.LocalizeKey("settings.legacy_migration", "Legacy Migration"), s.ImportProfileButton),
	))
	return container.NewTabItem(lang.LocalizeKey("settings.title", "Settings"), list), nil
}

func (s *Settings) importProfile() {
	path := s.state.ModInstallDir()
	if path == "" {
		dialog.ShowError(os.ErrNotExist, s.state.Window)
		return
	}

	modInstallLocation, err := os.OpenRoot(path)
	if err != nil {
		dialog.ShowError(err, s.state.Window)
		return
	}
	defer modInstallLocation.Close()

	installationInfo, err := modmgr.LoadInstallationInfo(modInstallLocation)
	if err != nil {
		dialog.ShowError(err, s.state.Window)
		return
	}

	entry := widget.NewEntry()
	entry.Validator = func(str string) error {
		if str == "" {
			return os.ErrInvalid
		}
		return nil
	}

	dialog.ShowForm(lang.LocalizeKey("profile.save_title", "Create Profile"), lang.LocalizeKey("common.save", "Save"), lang.LocalizeKey("common.cancel", "Cancel"), []*widget.FormItem{
		widget.NewFormItem(lang.LocalizeKey("profile.name", "Profile Name"), entry),
	}, func(confirm bool) {
		if !confirm {
			return
		}
		name := entry.Text
		mods := make(map[string]modmgr.ModVersion)
		for _, m := range installationInfo.InstalledMods {
			mods[m.ModVersion.ID] = m.ModVersion
		}

		prof := profile.Profile{
			ID:          uuid.New(),
			Name:        name,
			ModVersions: mods,
			LastUpdated: time.Now(),
		}

		if err := s.state.ProfileManager.Add(prof); err != nil {
			dialog.ShowError(err, s.state.Window)
			return
		}
		dialog.ShowInformation(lang.LocalizeKey("common.success", "Success"), lang.LocalizeKey("settings.profile_imported", "Profile imported successfully."), s.state.Window)
	}, s.state.Window)
}

func settingsEntry(title string, content fyne.CanvasObject) fyne.CanvasObject {
	label := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.New(layout.NewBorderLayout(nil, nil, label, nil), content, label)
}
