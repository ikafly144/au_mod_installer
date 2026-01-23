package modmgr

import (
	"fmt"
)

// VersionProvider is an interface to fetch mod version details.
type VersionProvider interface {
	GetModVersion(modID string, versionID string) (*ModVersion, error)
	GetLatestModVersion(modID string) (*ModVersion, error)
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
			if dep.Type != ModDependencyTypeRequired {
				continue
			}

			// Check if already resolved
			if _, ok := resolved[dep.ID]; ok {
				// TODO: Check version compatibility if dep.Version is specified
				continue
			}

			// Fetch dependency version
			// If dep.Version is empty, we might need a way to get the latest compatible version.
			// For now, if it's empty, we might have a problem unless the provider can handle it.
			// But usually ModDependency should have a version or we assume latest.
			// The current GetModVersion interface requires a versionID.
			
			var depVersion *ModVersion
			var err error
			if dep.Version == "" {
				depVersion, err = provider.GetLatestModVersion(dep.ID)
			} else {
				depVersion, err = provider.GetModVersion(dep.ID, dep.Version)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to fetch dependency %s (version: %s): %w", dep.ID, dep.Version, err)
			}

			resolved[dep.ID] = *depVersion
			queue = append(queue, *depVersion)
		}
	}

	return resolved, nil
}