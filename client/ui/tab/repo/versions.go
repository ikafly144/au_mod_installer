package repo

import (
	"fyne.io/fyne/v2/data/binding"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

// VersionSelect is an interface for version selection widgets.
type VersionSelect interface {
	GetVersions() []string
	SupplyMods(s func() ([]modmgr.ModVersion, error))
	AddListener(l binding.DataListener)
	SetSelected(version string)
	GetSelected() (string, error)
	Disable()
	Enable()
	Disabled() bool
	Refresh()
}
