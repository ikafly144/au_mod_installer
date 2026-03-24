package uicommon

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/ikafly144/au_mod_installer/common/rest/model"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
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

// Deprecated: TODO
func (s *State) InstallMods(modId string, versionData modmgr.ModVersion, progress progress.Progress) error {
	return errors.New("InstallMods is deprecated, use InstallModVersion instead")
}

func (s *State) resolveDependencies(modId string, versionData modmgr.ModVersion, resolved map[string]modmgr.ModVersion, unresolved map[string]struct{}, conflict map[string][]string) error {
	if versionData.ModID == "" {
		return fmt.Errorf("mod version %s has empty ModID", versionData.ID)
	}
	if _, ok := resolved[versionData.ID]; ok {
		return nil
	}
	if _, ok := unresolved[versionData.ID]; ok {
		return fmt.Errorf("circular dependency detected: %s", versionData.ID)
	}
	unresolved[versionData.ID] = struct{}{}
	for _, dep := range versionData.Dependencies {
		switch dep.DependencyType {
		case model.DependencyTypeRequired:
			depMod, err := s.ModVersion(dep.ModID, dep.VersionID)
			if err != nil {
				return fmt.Errorf("failed to resolve dependency %s version %s for mod %s: %w", dep.ModID, dep.VersionID, modId, err)
			}
			if err := s.resolveDependencies(dep.ModID, *depMod, resolved, unresolved, conflict); err != nil {
				return err
			}
		case model.DependencyTypeOptional:
			depMod, err := s.ModVersion(dep.ModID, dep.VersionID)
			if err != nil {
				slog.Info("Optional dependency not found, skipping", "dependency", dep.ModID, "mod", versionData.ID)
				continue
			}
			if err := s.resolveDependencies(dep.ModID, *depMod, resolved, unresolved, conflict); err != nil {
				return err
			}
		case model.DependencyTypeConflict:
			conflict[versionData.ID] = append(conflict[versionData.ID], dep.ModID)
		case model.DependencyTypeEmbedded:
			resolved[dep.ModID] = versionData
			delete(unresolved, dep.ModID)
		}
	}
	resolved[versionData.ID] = versionData
	delete(unresolved, versionData.ID)
	return nil
}
