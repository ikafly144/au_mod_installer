package common

import (
	"au_mod_installer/pkg/modmgr"
	"encoding/json"
	"log/slog"
	"net/http"
)

func (s *State) FetchMods() error {
	mods, err := ModProvider()
	if err != nil {
		return err
	}
	return s.Mods.Set(mods)
}

const modRepoURL = "https://cdn.sabafly.net/au_mods/mods_v4.json"

var ModProvider = func() ([]modmgr.Mod, error) {
	resp, err := http.Get(modRepoURL) // Pre-fetch to speed up later
	if err != nil {
		slog.Warn("Failed to pre-fetch mod repository", "error", err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		slog.Warn("Failed to pre-fetch mod repository", "status", resp.Status)
		return nil, err
	}
	defer resp.Body.Close()
	var mods []modmgr.Mod
	if err := json.NewDecoder(resp.Body).Decode(&mods); err != nil {
		slog.Warn("Failed to decode mod repository", "error", err)
		return nil, err
	}
	return mods, nil
}
