package rest

import (
	"context"
	"fmt"

	"github.com/ikafly144/modrepo/common/rest"
	"github.com/ikafly144/modrepo/pkg/modmgr"
	"github.com/ikafly144/modrepo/pkg/thunderstore"
)

type OfflineClient struct{}

var _ Client = (*OfflineClient)(nil)

func NewOfflineClient() *OfflineClient {
	return &OfflineClient{}
}

func (c *OfflineClient) ThunderstoreClient() *thunderstore.Client {
	return nil
}

func (c *OfflineClient) ServerBaseURL() string {
	return ""
}

func (c *OfflineClient) GetHealthStatus() (*rest.HealthStatus, error) {
	return &rest.HealthStatus{Status: "offline"}, nil
}

func (c *OfflineClient) GetVersionInfo() (*rest.VersionInfo, error) {
	return nil, fmt.Errorf("offline: version info not available")
}

func (c *OfflineClient) RefreshPackages(ctx context.Context) error {
	return nil
}

func (c *OfflineClient) GetModIDs(limit int, after string, before string) ([]string, error) {
	return nil, nil
}

func (c *OfflineClient) GetMod(modID string) (*modmgr.Mod, error) {
	return nil, fmt.Errorf("offline: mod not found: %s", modID)
}

func (c *OfflineClient) GetModVersionIDs(modID string, limit int, after string) ([]string, error) {
	return nil, nil
}

func (c *OfflineClient) GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error) {
	return nil, fmt.Errorf("offline: mod version not found: %s@%s", modID, versionID)
}

func (c *OfflineClient) GetLatestModVersion(modID string) (*modmgr.ModVersion, error) {
	return nil, fmt.Errorf("offline: latest version not found: %s", modID)
}

func (c *OfflineClient) GetModThumbnail(modID string) ([]byte, error) {
	return nil, fmt.Errorf("offline: thumbnail not available")
}

func (c *OfflineClient) CheckForUpdates(installedVersions map[string]string) (map[string]*modmgr.ModVersion, error) {
	return map[string]*modmgr.ModVersion{}, nil
}

func (c *OfflineClient) SearchMods(query string, category string, sortBy string) ([]*modmgr.Mod, error) {
	return nil, nil
}

func (c *OfflineClient) GetCategories() []string {
	return nil
}
