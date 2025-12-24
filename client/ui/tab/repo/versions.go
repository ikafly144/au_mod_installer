package repo

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

func newVersionSelectMenu(versions []modmgr.ModVersion) *versionSelectMenu {
	var versionStr []string
	for _, v := range versions {
		versionStr = append(versionStr, v.ID)
	}
	bind := binding.NewString()
	version := versionSelectMenu{
		versions:   versionStr,
		selected:   bind,
		selectMenu: widget.NewSelectWithData(versionStr, bind),
	}
	version.selectMenu.PlaceHolder = lang.LocalizeKey("repository.select_version", "バージョンを選択")
	return &version
}

type versionSelectMenu struct {
	versions   []string
	selectMenu *widget.Select
	selected   binding.String
}

func (v *versionSelectMenu) GetVersions() []string {
	if len(v.versions) == 0 {
		v.selectMenu.Disable()
		return []string{"N/A"}
	}
	v.selectMenu.Enable()
	return v.versions
}

func (v *versionSelectMenu) SupplyMods(s func() ([]modmgr.ModVersion, error), after func()) {
	go func() {
		mods, err := s()
		if err != nil {
			slog.Error("Failed to supply mod versions", "error", err)
			return
		}
		var vers []string
		for _, m := range mods {
			vers = append(vers, m.ID)
		}
		v.versions = vers
		fyne.DoAndWait(func() {
			v.selectMenu.SetOptions(v.GetVersions())
			if after != nil {
				after()
			}
		})
	}()
}

func (v *versionSelectMenu) Canvas() fyne.CanvasObject {
	return v.selectMenu
}

func (v *versionSelectMenu) AddListener(l binding.DataListener) {
	v.selected.AddListener(l)
}

func (v *versionSelectMenu) SetSelected(version string) {
	fyne.Do(func() {
		v.selectMenu.SetSelected(version)
	})
}

func (v *versionSelectMenu) GetSelected() (string, error) {
	return v.selected.Get()
}

func (v *versionSelectMenu) Disable() {
	if d, ok := v.Canvas().(fyne.Disableable); ok {
		d.Disable()
	}
}

func (v *versionSelectMenu) Enable() {
	if d, ok := v.Canvas().(fyne.Disableable); ok {
		d.Enable()
	}
}

func (v *versionSelectMenu) Disabled() bool {
	if d, ok := v.Canvas().(fyne.Disableable); ok {
		return d.Disabled()
	}
	return false
}

func (v *versionSelectMenu) Refresh() {
	v.Canvas().Refresh()
}
