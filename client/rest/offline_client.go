package rest

import (
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
	return []modmgr.Mod{}, nil
}

func (c *OfflineClient) GetMod(modID string) (*modmgr.Mod, error) {
	return nil, nil
}

func (c *OfflineClient) GetModVersions(modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	return []modmgr.ModVersion{}, nil
}

func (c *OfflineClient) GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error) {
	return nil, nil
}
