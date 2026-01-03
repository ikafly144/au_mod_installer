package settings

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/ikafly144/au_mod_installer/common/versioning"
)

type Settings struct {
	state        *uicommon.State
	BranchSelect *widget.Select
}

func NewSettings(state *uicommon.State) *Settings {
	branchOptions := []string{
		versioning.BranchStable.String(),
		versioning.BranchBeta.String(),
		versioning.BranchCanary.String(),
	}
	branchSelect := widget.NewSelect(branchOptions, nil)
	branchSelect.PlaceHolder = "Select Update Channel"
	branchSelect.OnChanged = func(s string) {
		fyne.CurrentApp().Preferences().SetString("core.update_branch", s)
	}
	currentBranch := fyne.CurrentApp().Preferences().StringWithFallback("core.update_branch", "stable")
	branchSelect.SetSelected(currentBranch)

	return &Settings{
		state:        state,
		BranchSelect: branchSelect,
	}
}

func (s *Settings) Tab() (*container.TabItem, error) {
	list := container.NewVScroll(container.NewVBox(
		settingsEntry(lang.LocalizeKey("settings.update_channel", "Update Channel"), s.BranchSelect),
	))
	return container.NewTabItem(lang.LocalizeKey("settings.title", "Settings"), list), nil
}

func settingsEntry(title string, content fyne.CanvasObject) fyne.CanvasObject {
	label := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.New(layout.NewBorderLayout(nil, nil, label, nil), content, label)
}
