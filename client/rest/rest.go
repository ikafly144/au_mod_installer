package rest

import (
	"github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

type Client interface {
	GetHealthStatus() (*rest.HealthStatus, error)
	GetModIDs(limit int, after string, before string) ([]string, error)
	GetMod(modID string) (*modmgr.Mod, error)
	GetModVersionIDs(modID string, limit int, after string) ([]string, error)
	GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error)
	GetLatestModVersion(modID string) (*modmgr.ModVersion, error)
	GetModThumbnail(modID string) ([]byte, error)
	CheckForUpdates(installedVersions map[string]string) (map[string]*modmgr.ModVersion, error)
}
