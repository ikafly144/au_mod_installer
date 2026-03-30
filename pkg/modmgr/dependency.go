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
	for _, m := range initialMods {
		resolved[m.ModID] = m
	}

	queue := make([]ModVersion, len(initialMods))
	copy(queue, initialMods)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dep := range current.Dependencies {
			if dep.DependencyType != ModDependencyTypeRequired {
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
				return nil, err
			}

			resolved[dep.ModID] = *depVersion
			queue = append(queue, *depVersion)
		}
	}

	return resolved, nil
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
	constraint := strings.TrimSpace(dep.VersionID)
	if constraint == "" || strings.EqualFold(constraint, "any") {
		return nil
	}

	if isExactVersionConstraint(constraint) {
		if resolvedDep.ID != constraint {
			return fmt.Errorf("version conflict for mod %s: required %s but resolved %s", dep.ModID, dep.VersionID, resolvedDep.ID)
		}
		return nil
	}

	if strings.EqualFold(constraint, "latest") {
		latest, err := provider.GetLatestModVersion(dep.ModID)
		if err != nil {
			return fmt.Errorf("failed to fetch latest dependency %s: %w", dep.ModID, err)
		}
		if latest == nil {
			return fmt.Errorf("failed to fetch latest dependency %s: version not found", dep.ModID)
		}
		if resolvedDep.ID != latest.ID {
			return fmt.Errorf("version conflict for mod %s: required latest (%s) but resolved %s", dep.ModID, latest.ID, resolvedDep.ID)
		}
		return nil
	}

	if !version.NewConstrainGroupFromString(constraint).Match(resolvedDep.ID) {
		return fmt.Errorf("version conflict for mod %s: required %s but resolved %s", dep.ModID, dep.VersionID, resolvedDep.ID)
	}
	return nil
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
