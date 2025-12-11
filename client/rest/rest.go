package rest

import "github.com/ikafly144/au_mod_installer/pkg/modmgr"

type Client interface {
	GetModList(limit int, after string, before string) ([]modmgr.Mod, error)
	GetMod(modID string) (*modmgr.Mod, error)
	GetModVersions(modID string, limit int, after string) ([]modmgr.ModVersion, error)
	GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error)
}
