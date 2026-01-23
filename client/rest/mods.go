package rest

import (
	"fmt"
	"net/url"

	"github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

func (c *clientImpl) GetModList(limit int, after string, before string) ([]modmgr.Mod, error) {
	var mods []modmgr.Mod

	values := make(url.Values)
	if limit > 0 {
		values.Set("limit", fmt.Sprint(limit))
	}
	if after != "" {
		values.Set("after", after)
	}
	if before != "" {
		values.Set("before", before)
	}

	err := c.do(rest.EndpointGetModList.Compile(values, nil), nil, &mods, 1)
	return mods, err
}

func (c *clientImpl) GetMod(modID string) (*modmgr.Mod, error) {
	var mod modmgr.Mod
	err := c.do(rest.EndpointGetModDetails.Compile(nil, modID), nil, &mod, 1)
	return &mod, err
}

func (c *clientImpl) GetModVersions(modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	var versions []modmgr.ModVersion
	err := c.do(rest.EndpointGetModVersions.Compile(nil, modID), nil, &versions, 1)
	return versions, err
}

func (c *clientImpl) GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error) {
	var modVersion modmgr.ModVersion
	err := c.do(rest.EndpointGetModVersion.Compile(nil, modID, versionID), nil, &modVersion, 1)
	return &modVersion, err
}

func (c *clientImpl) GetLatestModVersion(modID string) (*modmgr.ModVersion, error) {
	mod, err := c.GetMod(modID)
	if err != nil {
		return nil, err
	}
	if mod.LatestVersion == "" {
		return nil, fmt.Errorf("mod %s does not have a latest version", modID)
	}
	return c.GetModVersion(modID, mod.LatestVersion)
}
