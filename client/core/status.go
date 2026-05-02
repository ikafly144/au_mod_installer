package core

import (
	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
)

func (a *App) GetGameVersion(gamePath string) (string, error) {
	return aumgr.GetVersion(gamePath)
}
