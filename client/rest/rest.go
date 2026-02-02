package rest

import (
	"github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

type Client interface {
	GetHealthStatus() (*rest.HealthStatus, error)
	GetModList(limit int, after string, before string) ([]modmgr.Mod, error)
	GetMod(modID string) (*modmgr.Mod, error)
	GetModVersions(modID string, limit int, after string) ([]modmgr.ModVersion, error)
	GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error)
	GetLatestModVersion(modID string) (*modmgr.ModVersion, error)
	CheckForUpdates(installedVersions map[string]string) (map[string]*modmgr.ModVersion, error)
}
