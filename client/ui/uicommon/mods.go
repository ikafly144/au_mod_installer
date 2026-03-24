package uicommon

import (
	"fmt"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

func (s *State) Mod(id string) (*modmgr.Mod, error) {
	return s.Rest.GetMod(id)
}

func (s *State) ModVersion(modId, versionId string) (*modmgr.ModVersion, error) {
	if versionId == "" {
		mod, err := s.Mod(modId)
		if err != nil {
			return nil, err
		}
		if mod.LatestVersionID == "" {
			return nil, fmt.Errorf("mod %s has no latest version", modId)
		}
		versionId = mod.LatestVersionID
	}
	return s.Rest.GetModVersion(modId, versionId)
}
