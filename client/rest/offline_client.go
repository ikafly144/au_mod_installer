package rest

import (
	"errors"

	"github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

type OfflineClient struct{}

var _ Client = (*OfflineClient)(nil)

func NewOfflineClient() *OfflineClient {
	return &OfflineClient{}
}

func (c *OfflineClient) GetHealthStatus() (*rest.HealthStatus, error) {
	return &rest.HealthStatus{
		Status: "offline",
	}, nil
}

func (c *OfflineClient) GetModList(limit int, after string, before string) ([]modmgr.Mod, error) {
	return nil, errors.New("offline mode: mod list not available")
}

func (c *OfflineClient) GetMod(modID string) (*modmgr.Mod, error) {
	return nil, errors.New("offline mode: mod details not available")
}

func (c *OfflineClient) GetModVersions(modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	return nil, errors.New("offline mode: mod versions not available")
}

func (c *OfflineClient) GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error) {
	return nil, errors.New("offline mode: mod version details not available")
}

func (c *OfflineClient) GetLatestModVersion(modID string) (*modmgr.ModVersion, error) {
	return nil, errors.New("offline mode: latest mod version details not available")
}

func (c *OfflineClient) CheckForUpdates(installedVersions map[string]string) (map[string]*modmgr.ModVersion, error) {
	return nil, errors.New("offline mode: update check not available")
}
