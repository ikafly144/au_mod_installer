package rest

import (
	"context"

	"github.com/ikafly144/modrepo/common/rest"
	"github.com/ikafly144/modrepo/pkg/modmgr"
	"github.com/ikafly144/modrepo/pkg/thunderstore"
)

type Client interface {
	ServerBaseURL() string
	GetHealthStatus() (*rest.HealthStatus, error)
	GetVersionInfo() (*rest.VersionInfo, error)
	GetModIDs(limit int, after string, before string) ([]string, error)
	GetMod(modID string) (*modmgr.Mod, error)
	GetModVersionIDs(modID string, limit int, after string) ([]string, error)
	GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error)
	GetLatestModVersion(modID string) (*modmgr.ModVersion, error)
	GetModThumbnail(modID string) ([]byte, error)
	CheckForUpdates(installedVersions map[string]string) (map[string]*modmgr.ModVersion, error)
	SearchMods(query string, category string, sortBy string) ([]*modmgr.Mod, error)
	GetCategories() []string
	RefreshPackages(ctx context.Context) error
	ThunderstoreClient() *thunderstore.Client
}

func NewClient(serverURL string) Client {
	return NewThunderstoreClient("", nil)
}
