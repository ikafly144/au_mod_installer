package rest

import (
	"fmt"
	"net/url"

	"github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/common/rest/model"
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
	var mod model.ModDetails
	err := c.do(rest.EndpointGetModDetail.Compile(nil, modID), nil, &mod, 1)
	return &modmgr.Mod{ModDetails: mod}, err
}

func (c *clientImpl) GetModVersionIDs(modID string, limit int, after string) ([]string, error) {
	var versions model.ModVersionListResult
	err := c.do(rest.EndpointGetModVersionList.Compile(nil, modID), nil, &versions, 1)
	return versions.IDs, err
}

func (c *clientImpl) GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error) {
	var modVersion modmgr.ModVersion
	err := c.do(rest.EndpointGetModVersionDetail.Compile(nil, modID, versionID), nil, &modVersion, 1)
	return &modVersion, err
}

func (c *clientImpl) GetLatestModVersion(modID string) (*modmgr.ModVersion, error) {
	mod, err := c.GetMod(modID)
	if err != nil {
		return nil, err
	}
	if mod.LatestVersionID == "" {
		return nil, fmt.Errorf("mod %s does not have a latest version", modID)
	}
	return c.GetModVersion(modID, mod.LatestVersionID)
}

func (c *clientImpl) CheckForUpdates(installedVersions map[string]string) (map[string]*modmgr.ModVersion, error) {
	updates := make(map[string]*modmgr.ModVersion)
	for modID, currentVersion := range installedVersions {
		latest, err := c.GetLatestModVersion(modID)
		if err != nil {
			continue
		}
		if latest != nil && latest.ID != currentVersion {
			updates[modID] = latest
		}
	}
	return updates, nil
}
