package uicommon

import (
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
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
		if mod.LatestVersion == "" {
			return nil, fmt.Errorf("mod %s has no latest version", modId)
		}
		versionId = mod.LatestVersion
	}
	return s.Rest.GetModVersion(modId, versionId)
}

func (s *State) InstallMods(modId string, versionData modmgr.ModVersion, progress progress.Progress) error {
	if !s.installLock.TryLock() {
		slog.Warn("Mod installation already in progress")
		return nil
	}
	defer s.installLock.Unlock()
	slog.Info("Starting mod installation", "modId", modId, "versionId", versionData.ID)

	var versions []modmgr.ModVersion
	if len(versionData.Mods) > 0 {
		for _, modPack := range versionData.Mods {
			modVersion, err := s.ModVersion(modPack.ID, modPack.Version)
			if err != nil {
				slog.Error("Failed to get mod version from mod pack", "modPackId", modPack.ID, "version", modPack.Version, "error", err)
				return err
			}
			versions = append(versions, *modVersion)
		}
	} else {
		versions = append(versions, versionData)
	}

	resolved := make(map[string]modmgr.ModVersion)
	unresolved := make(map[string]struct{})
	conflict := make(map[string][]string)
	for _, v := range versions {
		if err := s.resolveDependencies(modId, v, resolved, unresolved, conflict); err != nil {
			slog.Error("Failed to resolve dependencies", "modId", modId, "versionId", v.ID, "error", err)
			return err
		}
	}
	slog.Info("Resolved dependencies", "mods", fmt.Sprintf("%v", resolved))

	if len(conflict) > 0 {
		for mod, conflicts := range conflict {
			slog.Warn("Mod has conflicts", "mod", mod, "conflicts", conflicts)
		}
		return fmt.Errorf("conflicting mods detected: %v", conflict)
	}
	path, err := s.SelectedGamePath.Get()
	if err != nil {
		slog.Error("Failed to get selected game path", "error", err)
		return err
	}
	gameRoot, err := os.OpenRoot(path)
	if err != nil {
		slog.Error("Failed to open game root", "error", err)
		return err
	}

	launcherType := aumgr.DetectLauncherType(path)
	gameVersion, err := aumgr.GetVersion(path)
	if err != nil {
		slog.Error("Failed to get game version", "error", err)
		return err
	}

	binaryType, err := aumgr.GetBinaryType(path)
	if err != nil {
		slog.Error("Failed to detect binary type", "error", err)
		return err
	}
	slog.Info("Detected game binary type", "type", binaryType)

	installVersions := make([]modmgr.ModVersion, 0, len(resolved))
	for _, v := range resolved {
		if slices.ContainsFunc(installVersions, func(x modmgr.ModVersion) bool {
			return x.ID == v.ID && x.ModID == v.ModID
		}) {
			continue
		}
		if !v.IsCompatible(launcherType, binaryType, gameVersion) {
			slog.Warn("Mod version is not compatible, skipping", "modId", v.ID, "versionId", v.ID)
			return fmt.Errorf("mod %s version %s is not compatible with version %s", modId, v.ID, gameVersion)
		}
		installVersions = append(installVersions, v)
	}
	if _, err := modmgr.InstallMod(gameRoot, gameVersion, launcherType, binaryType, installVersions, progress); err != nil {
		slog.Error("Mod installation failed", "error", err)
		return err
	}
	return nil
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
		switch dep.Type {
		case modmgr.ModDependencyTypeRequired:
			depMod, err := s.ModVersion(dep.ID, dep.Version)
			if err != nil {
				return fmt.Errorf("failed to resolve dependency %s version %s for mod %s: %w", dep.ID, dep.Version, modId, err)
			}
			if err := s.resolveDependencies(dep.ID, *depMod, resolved, unresolved, conflict); err != nil {
				return err
			}
		case modmgr.ModDependencyTypeOptional:
			depMod, err := s.ModVersion(dep.ID, dep.Version)
			if err != nil {
				slog.Info("Optional dependency not found, skipping", "dependency", dep.ID, "mod", versionData.ID)
				continue
			}
			if err := s.resolveDependencies(dep.ID, *depMod, resolved, unresolved, conflict); err != nil {
				return err
			}
		case modmgr.ModDependencyTypeConflict:
			conflict[versionData.ID] = append(conflict[versionData.ID], dep.ID)
		case modmgr.ModDependencyTypeEmbedded:
			resolved[dep.ID] = versionData
			delete(unresolved, dep.ID)
		}
	}
	resolved[versionData.ID] = versionData
	delete(unresolved, versionData.ID)
	return nil
}
