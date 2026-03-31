package modmgr

import (
	"fmt"
	"strings"

	version "github.com/mcuadros/go-version"

	"github.com/ikafly144/au_mod_installer/common/rest/model"
)

// VersionProvider is an interface to fetch mod version details.
type VersionProvider interface {
	GetModVersion(modID string, versionID string) (*ModVersion, error)
	GetLatestModVersion(modID string) (*ModVersion, error)
	GetModVersionIDs(modID string, limit int, after string) ([]string, error)
}

// ResolveDependencies recursively finds all required dependencies for a given set of mod versions.
// It returns a map of ModID to ModVersion containing all original mods and their required dependencies.
func ResolveDependencies(initialMods []ModVersion, provider VersionProvider) (map[string]ModVersion, error) {
	resolved := make(map[string]ModVersion)
	failedRequired := make(map[string]error)
	for _, m := range initialMods {
		resolved[m.ModID] = m
	}

	queue := make([]ModVersion, len(initialMods))
	copy(queue, initialMods)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dep := range current.Dependencies {
			depType, err := normalizeDependencyType(dep.DependencyType)
			if err != nil {
				return nil, fmt.Errorf("invalid dependency type for %s@%s -> %s: %w", current.ModID, current.ID, dep.ModID, err)
			}

			if depType != ModDependencyTypeRequired {
				continue
			}

			embeddedProviders, err := collectEmbeddedProviders(resolved)
			if err != nil {
				return nil, err
			}
			if len(embeddedProviders[dep.ModID]) > 0 {
				// This required dependency is satisfied by another resolved mod that embeds it.
				continue
			}

			// Check if already resolved
			if resolvedDep, ok := resolved[dep.ModID]; ok {
				if err := checkDependencyConstraint(resolvedDep, dep, provider); err != nil {
					return nil, err
				}
				continue
			}

			depVersion, err := resolveDependencyVersion(provider, dep)
			if err != nil {
				failedRequired[requiredFailureKey(current, dep)] = err
				continue
			}

			delete(failedRequired, requiredFailureKey(current, dep))
			resolved[dep.ModID] = *depVersion
			queue = append(queue, *depVersion)
		}
	}

	if err := validateResolvedDependencies(resolved, provider, failedRequired); err != nil {
		return nil, err
	}

	return resolved, nil
}

func validateResolvedDependencies(resolved map[string]ModVersion, provider VersionProvider, failedRequired map[string]error) error {
	embeddedProviders, err := collectEmbeddedProviders(resolved)
	if err != nil {
		return err
	}

	for _, current := range resolved {
		for _, dep := range current.Dependencies {
			depType, err := normalizeDependencyType(dep.DependencyType)
			if err != nil {
				return fmt.Errorf("invalid dependency type for %s@%s -> %s: %w", current.ModID, current.ID, dep.ModID, err)
			}

			resolvedDep, found := resolved[dep.ModID]
			switch depType {
			case ModDependencyTypeRequired:
				if !found && len(embeddedProviders[dep.ModID]) == 0 {
					if resolveErr, ok := failedRequired[requiredFailureKey(current, dep)]; ok {
						return resolveErr
					}
					return fmt.Errorf("required dependency not resolved for mod %s: %s", current.ModID, dep.ModID)
				}
				if found {
					if err := checkDependencyConstraint(resolvedDep, dep, provider); err != nil {
						return err
					}
				}
			case ModDependencyTypeOptional:
				if found {
					if err := checkOptionalDependencyConstraint(resolvedDep, dep, provider); err != nil {
						return err
					}
				}
			case ModDependencyTypeConflict:
				if found {
					if err := checkConflictDependencyConstraint(resolvedDep, dep, provider); err != nil {
						return err
					}
				}
			case ModDependencyTypeEmbedded:
				// Embedded means the dependency is bundled by the current mod.
				// It can coexist in resolved set (e.g. explicitly selected), so no error here.
			}
		}
	}

	return nil
}

func collectEmbeddedProviders(mods map[string]ModVersion) (map[string][]string, error) {
	embeddedProviders := make(map[string][]string)
	for _, current := range mods {
		for _, dep := range current.Dependencies {
			depType, err := normalizeDependencyType(dep.DependencyType)
			if err != nil {
				return nil, fmt.Errorf("invalid dependency type for %s@%s -> %s: %w", current.ModID, current.ID, dep.ModID, err)
			}
			if depType == ModDependencyTypeEmbedded {
				embeddedProviders[dep.ModID] = append(embeddedProviders[dep.ModID], current.ModID)
			}
		}
	}
	return embeddedProviders, nil
}

func resolveDependencyVersion(provider VersionProvider, dep model.ModVersionDependency) (*ModVersion, error) {
	constraint := strings.TrimSpace(dep.VersionID)
	if constraint == "" || strings.EqualFold(constraint, "any") || strings.EqualFold(constraint, "latest") {
		depVersion, err := provider.GetLatestModVersion(dep.ModID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch dependency %s (version: %s): %w", dep.ModID, dep.VersionID, err)
		}
		if depVersion == nil {
			return nil, fmt.Errorf("failed to fetch dependency %s (version: %s): version not found", dep.ModID, dep.VersionID)
		}
		return depVersion, nil
	}

	if isExactVersionConstraint(constraint) {
		depVersion, err := provider.GetModVersion(dep.ModID, constraint)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch dependency %s (version: %s): %w", dep.ModID, dep.VersionID, err)
		}
		if depVersion == nil {
			return nil, fmt.Errorf("failed to fetch dependency %s (version: %s): version not found", dep.ModID, dep.VersionID)
		}
		return depVersion, nil
	}

	versionIDs, err := provider.GetModVersionIDs(dep.ModID, 100, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list versions for dependency %s: %w", dep.ModID, err)
	}

	matchedVersionID, found := bestMatchedVersionID(versionIDs, constraint)
	if !found {
		return nil, fmt.Errorf("failed to fetch dependency %s (version: %s): no matching version found", dep.ModID, dep.VersionID)
	}

	depVersion, err := provider.GetModVersion(dep.ModID, matchedVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dependency %s (version: %s): %w", dep.ModID, dep.VersionID, err)
	}
	if depVersion == nil {
		return nil, fmt.Errorf("failed to fetch dependency %s (version: %s): version not found", dep.ModID, dep.VersionID)
	}

	return depVersion, nil
}

func checkDependencyConstraint(resolvedDep ModVersion, dep model.ModVersionDependency, provider VersionProvider) error {
	matched, requiredVersion, err := matchesDependencyConstraint(resolvedDep, dep, provider)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("version conflict for mod %s: required %s but resolved %s", dep.ModID, requiredVersion, resolvedDep.ID)
	}
	return nil
}

func checkOptionalDependencyConstraint(resolvedDep ModVersion, dep model.ModVersionDependency, provider VersionProvider) error {
	matched, requiredVersion, err := matchesDependencyConstraint(resolvedDep, dep, provider)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("optional dependency version conflict for mod %s: optional %s but resolved %s", dep.ModID, requiredVersion, resolvedDep.ID)
	}
	return nil
}

func checkConflictDependencyConstraint(resolvedDep ModVersion, dep model.ModVersionDependency, provider VersionProvider) error {
	matched, conflictVersion, err := matchesDependencyConstraint(resolvedDep, dep, provider)
	if err != nil {
		return err
	}
	if matched {
		return fmt.Errorf("dependency conflict for mod %s: conflicted %s and resolved %s", dep.ModID, conflictVersion, resolvedDep.ID)
	}
	return nil
}

func matchesDependencyConstraint(resolvedDep ModVersion, dep model.ModVersionDependency, provider VersionProvider) (bool, string, error) {
	constraint := strings.TrimSpace(dep.VersionID)
	if constraint == "" || strings.EqualFold(constraint, "any") {
		return true, "any", nil
	}

	if isExactVersionConstraint(constraint) {
		return resolvedDep.ID == constraint, dep.VersionID, nil
	}

	if strings.EqualFold(constraint, "latest") {
		latest, err := provider.GetLatestModVersion(dep.ModID)
		if err != nil {
			return false, "", fmt.Errorf("failed to fetch latest dependency %s: %w", dep.ModID, err)
		}
		if latest == nil {
			return false, "", fmt.Errorf("failed to fetch latest dependency %s: version not found", dep.ModID)
		}
		return resolvedDep.ID == latest.ID, fmt.Sprintf("latest (%s)", latest.ID), nil
	}

	return version.NewConstrainGroupFromString(constraint).Match(resolvedDep.ID), dep.VersionID, nil
}

func normalizeDependencyType(depType model.DependencyType) (ModDependencyType, error) {
	switch strings.ToLower(strings.TrimSpace(string(depType))) {
	case "", string(ModDependencyTypeRequired):
		return ModDependencyTypeRequired, nil
	case string(ModDependencyTypeOptional):
		return ModDependencyTypeOptional, nil
	case string(ModDependencyTypeConflict):
		return ModDependencyTypeConflict, nil
	case string(ModDependencyTypeEmbedded):
		return ModDependencyTypeEmbedded, nil
	default:
		return "", fmt.Errorf("unknown dependency type %q", depType)
	}
}

func requiredFailureKey(current ModVersion, dep model.ModVersionDependency) string {
	return fmt.Sprintf("%s@%s->%s@%s", current.ModID, current.ID, dep.ModID, dep.VersionID)
}

func isExactVersionConstraint(constraint string) bool {
	return !strings.ContainsAny(constraint, "<>!=~*xX,^@,") && !strings.EqualFold(constraint, "latest") && !strings.EqualFold(constraint, "any")
}

func bestMatchedVersionID(versionIDs []string, constraint string) (string, bool) {
	group := version.NewConstrainGroupFromString(constraint)
	best := ""
	for _, versionID := range versionIDs {
		if !group.Match(versionID) {
			continue
		}
		if best == "" || compareVersionID(versionID, best) > 0 {
			best = versionID
		}
	}
	return best, best != ""
}

func compareVersionID(a, b string) int {
	if cmp := version.CompareSimple(version.Normalize(a), version.Normalize(b)); cmp != 0 {
		return cmp
	}
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}
