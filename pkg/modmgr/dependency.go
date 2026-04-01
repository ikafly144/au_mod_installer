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
// The algorithm attempts to find the best compatible version combination, trying newer versions first
// but backtracking to older versions if conflicts arise.
// When multiple MODs require the same dependency, it merges constraints to find a version that satisfies all.
func ResolveDependencies(initialMods []ModVersion, provider VersionProvider) (map[string]ModVersion, error) {
	resolved := make(map[string]ModVersion)
	for _, m := range initialMods {
		resolved[m.ModID] = m
	}

	// Collect all required dependencies and their constraints first
	allDeps := collectAllRequiredDependencies(initialMods, resolved)

	// Resolve dependencies with merged constraints
	if err := resolveDependenciesWithConstraints(resolved, allDeps, provider); err != nil {
		return nil, err
	}

	// Final validation
	if err := validateResolvedDependencies(resolved, provider, nil); err != nil {
		return nil, err
	}

	return resolved, nil
}

// dependencyConstraints holds all constraints for a single dependency
type dependencyConstraints struct {
	modID       string
	constraints []model.ModVersionDependency
}

// collectAllRequiredDependencies recursively collects all required dependencies and their constraints
func collectAllRequiredDependencies(queue []ModVersion, alreadyResolved map[string]ModVersion) map[string]*dependencyConstraints {
	allDeps := make(map[string]*dependencyConstraints)
	visited := make(map[string]bool)

	// Mark already resolved as visited, but still collect their dependencies
	for _, mod := range alreadyResolved {
		visited[mod.ModID] = true
	}

	var collectRecursive func(mods []ModVersion)
	collectRecursive = func(mods []ModVersion) {
		for _, current := range mods {
			wasVisited := visited[current.ModID]
			visited[current.ModID] = true

			for _, dep := range current.Dependencies {
				depType, err := normalizeDependencyType(dep.DependencyType)
				if err != nil || depType != ModDependencyTypeRequired {
					continue
				}

				// Skip if this dependency is already resolved
				if _, alreadyResolved := alreadyResolved[dep.ModID]; alreadyResolved {
					continue
				}

				if allDeps[dep.ModID] == nil {
					allDeps[dep.ModID] = &dependencyConstraints{
						modID:       dep.ModID,
						constraints: []model.ModVersionDependency{},
					}
				}
				allDeps[dep.ModID].constraints = append(allDeps[dep.ModID].constraints, dep)
			}

			// Don't recurse if we already visited this node
			if wasVisited {
				continue
			}
		}
	}

	collectRecursive(queue)
	return allDeps
}

// resolveDependenciesWithConstraints resolves dependencies considering all constraints
func resolveDependenciesWithConstraints(resolved map[string]ModVersion, allDeps map[string]*dependencyConstraints, provider VersionProvider) error {
	// Keep resolving until no more dependencies to add
	for len(allDeps) > 0 {
		// Pick one dependency to resolve
		var depConstraint *dependencyConstraints
		for _, dc := range allDeps {
			depConstraint = dc
			break
		}

		embeddedProviders, err := collectEmbeddedProviders(resolved)
		if err != nil {
			return err
		}
		if len(embeddedProviders[depConstraint.modID]) > 0 {
			// This dependency is satisfied by an embedded provider
			delete(allDeps, depConstraint.modID)
			continue
		}

		// Get candidates that satisfy ALL constraints
		candidates, err := getCandidateVersionsForConstraints(provider, depConstraint.constraints)
		if err != nil {
			return err
		}

		if len(candidates) == 0 {
			constraintStrs := make([]string, len(depConstraint.constraints))
			for i, c := range depConstraint.constraints {
				constraintStrs[i] = c.VersionID
			}
			return fmt.Errorf("failed to find version for %s satisfying all constraints: %v", depConstraint.modID, constraintStrs)
		}

		// Try each candidate (ordered newest to oldest)
		var lastErr error
		resolved_any := false
		for _, candidate := range candidates {
			// Create a test resolution
			testResolved := make(map[string]ModVersion)
			for k, v := range resolved {
				testResolved[k] = v
			}
			testResolved[depConstraint.modID] = *candidate

			// Collect new dependencies introduced by this candidate
			// Pass only the original resolved (not testResolved) to avoid skipping new deps
			newDeps := collectAllRequiredDependencies([]ModVersion{*candidate}, resolved)

			// Merge with remaining dependencies
			testAllDeps := make(map[string]*dependencyConstraints)
			for k, v := range allDeps {
				if k == depConstraint.modID {
					continue // Skip the one we just resolved
				}
				// Deep copy the constraints
				testAllDeps[k] = &dependencyConstraints{
					modID:       v.modID,
					constraints: append([]model.ModVersionDependency{}, v.constraints...),
				}
			}
			for k, v := range newDeps {
				if testAllDeps[k] == nil {
					testAllDeps[k] = v
				} else {
					// Merge constraints
					testAllDeps[k].constraints = append(testAllDeps[k].constraints, v.constraints...)
				}
			}

			// Try to resolve the rest
			if err := resolveDependenciesWithConstraints(testResolved, testAllDeps, provider); err != nil {
				lastErr = err
				continue
			}

			// Success! Update resolved
			for k, v := range testResolved {
				resolved[k] = v
			}
			resolved_any = true
			break
		}

		if !resolved_any {
			if lastErr != nil {
				return lastErr
			}
			return fmt.Errorf("failed to resolve dependency %s with any candidate version", depConstraint.modID)
		}

		// Remove this dependency from the list
		delete(allDeps, depConstraint.modID)
	}

	return nil
}

// getCandidateVersionsForConstraints returns versions that satisfy ALL given constraints
func getCandidateVersionsForConstraints(provider VersionProvider, constraints []model.ModVersionDependency) ([]*ModVersion, error) {
	if len(constraints) == 0 {
		return nil, fmt.Errorf("no constraints provided")
	}

	modID := constraints[0].ModID

	// Check if all constraints are exact and identical
	firstConstraint := strings.TrimSpace(constraints[0].VersionID)
	allSame := true
	for _, c := range constraints {
		if strings.TrimSpace(c.VersionID) != firstConstraint {
			allSame = false
			break
		}
	}

	if allSame {
		// All constraints are the same, use single constraint logic
		return getCandidateVersions(provider, constraints[0])
	}

	// Get all available versions
	versionIDs, err := provider.GetModVersionIDs(modID, 100, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list versions for dependency %s: %w", modID, err)
	}

	// Filter versions that satisfy ALL constraints
	matchingVersionIDs := make([]string, 0)
	for _, versionID := range versionIDs {
		satisfiesAll := true
		for _, constraint := range constraints {
			constraintStr := strings.TrimSpace(constraint.VersionID)

			// Check if this version satisfies this constraint
			if constraintStr == "" || strings.EqualFold(constraintStr, "any") {
				continue // "any" is always satisfied
			}

			if strings.EqualFold(constraintStr, "latest") {
				// For "latest", we'll check later
				continue
			}

			if isExactVersionConstraint(constraintStr) {
				if versionID != constraintStr {
					satisfiesAll = false
					break
				}
			} else {
				group := version.NewConstrainGroupFromString(constraintStr)
				if !group.Match(versionID) {
					satisfiesAll = false
					break
				}
			}
		}

		if satisfiesAll {
			matchingVersionIDs = append(matchingVersionIDs, versionID)
		}
	}

	// Handle "latest" constraints
	hasLatestConstraint := false
	for _, constraint := range constraints {
		if strings.EqualFold(strings.TrimSpace(constraint.VersionID), "latest") {
			hasLatestConstraint = true
			break
		}
	}

	if hasLatestConstraint && len(matchingVersionIDs) > 0 {
		// If there's a "latest" constraint, only return the latest matching version
		latest, err := provider.GetLatestModVersion(modID)
		if err != nil {
			return nil, err
		}
		if latest != nil {
			// Check if latest is in our matching list
			for _, vid := range matchingVersionIDs {
				if vid == latest.VersionID {
					return []*ModVersion{latest}, nil
				}
			}
		}
		// Latest doesn't satisfy other constraints
		return nil, nil
	}

	// Sort matching versions (newest first)
	for i := 0; i < len(matchingVersionIDs)-1; i++ {
		for j := i + 1; j < len(matchingVersionIDs); j++ {
			if compareVersionID(matchingVersionIDs[i], matchingVersionIDs[j]) < 0 {
				matchingVersionIDs[i], matchingVersionIDs[j] = matchingVersionIDs[j], matchingVersionIDs[i]
			}
		}
	}

	// Fetch version objects
	candidates := make([]*ModVersion, 0, len(matchingVersionIDs))
	for _, versionID := range matchingVersionIDs {
		depVersion, err := provider.GetModVersion(modID, versionID)
		if err != nil {
			continue
		}
		if depVersion != nil {
			candidates = append(candidates, depVersion)
		}
	}

	return candidates, nil
}

// getCandidateVersions returns a list of candidate versions in priority order (best to worst)
func getCandidateVersions(provider VersionProvider, dep model.ModVersionDependency) ([]*ModVersion, error) {
	constraint := strings.TrimSpace(dep.VersionID)

	// Handle "any", "latest", or empty constraint
	if constraint == "" || strings.EqualFold(constraint, "any") || strings.EqualFold(constraint, "latest") {
		depVersion, err := provider.GetLatestModVersion(dep.ModID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch dependency %s (version: %s): %w", dep.ModID, dep.VersionID, err)
		}
		if depVersion == nil {
			return nil, fmt.Errorf("failed to fetch dependency %s (version: %s): version not found", dep.ModID, dep.VersionID)
		}
		return []*ModVersion{depVersion}, nil
	}

	// Handle exact version constraint
	if isExactVersionConstraint(constraint) {
		depVersion, err := provider.GetModVersion(dep.ModID, constraint)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch dependency %s (version: %s): %w", dep.ModID, dep.VersionID, err)
		}
		if depVersion == nil {
			return nil, fmt.Errorf("failed to fetch dependency %s (version: %s): version not found", dep.ModID, dep.VersionID)
		}
		return []*ModVersion{depVersion}, nil
	}

	// Handle range constraint - get all matching versions in descending order
	versionIDs, err := provider.GetModVersionIDs(dep.ModID, 100, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list versions for dependency %s: %w", dep.ModID, err)
	}

	matchingVersionIDs := getAllMatchingVersionIDs(versionIDs, constraint)
	if len(matchingVersionIDs) == 0 {
		return nil, nil
	}

	// Fetch all matching versions
	candidates := make([]*ModVersion, 0, len(matchingVersionIDs))
	for _, versionID := range matchingVersionIDs {
		depVersion, err := provider.GetModVersion(dep.ModID, versionID)
		if err != nil {
			continue // Skip versions that fail to fetch
		}
		if depVersion != nil {
			candidates = append(candidates, depVersion)
		}
	}

	return candidates, nil
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
				return fmt.Errorf("invalid dependency type for %s@%s -> %s: %w", current.ModID, current.VersionID, dep.ModID, err)
			}

			resolvedDep, found := resolved[dep.ModID]
			switch depType {
			case ModDependencyTypeRequired:
				if !found && len(embeddedProviders[dep.ModID]) == 0 {
					if failedRequired != nil {
						if resolveErr, ok := failedRequired[requiredFailureKey(current, dep)]; ok {
							return resolveErr
						}
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
				return nil, fmt.Errorf("invalid dependency type for %s@%s -> %s: %w", current.ModID, current.VersionID, dep.ModID, err)
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
		return fmt.Errorf("version conflict for mod %s: required %s but resolved %s", dep.ModID, requiredVersion, resolvedDep.VersionID)
	}
	return nil
}

func checkOptionalDependencyConstraint(resolvedDep ModVersion, dep model.ModVersionDependency, provider VersionProvider) error {
	matched, requiredVersion, err := matchesDependencyConstraint(resolvedDep, dep, provider)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("optional dependency version conflict for mod %s: optional %s but resolved %s", dep.ModID, requiredVersion, resolvedDep.VersionID)
	}
	return nil
}

func checkConflictDependencyConstraint(resolvedDep ModVersion, dep model.ModVersionDependency, provider VersionProvider) error {
	matched, conflictVersion, err := matchesDependencyConstraint(resolvedDep, dep, provider)
	if err != nil {
		return err
	}
	if matched {
		return fmt.Errorf("dependency conflict for mod %s: conflicted %s and resolved %s", dep.ModID, conflictVersion, resolvedDep.VersionID)
	}
	return nil
}

func matchesDependencyConstraint(resolvedDep ModVersion, dep model.ModVersionDependency, provider VersionProvider) (bool, string, error) {
	constraint := strings.TrimSpace(dep.VersionID)
	if constraint == "" || strings.EqualFold(constraint, "any") {
		return true, "any", nil
	}

	if isExactVersionConstraint(constraint) {
		return resolvedDep.VersionID == constraint, dep.VersionID, nil
	}

	if strings.EqualFold(constraint, "latest") {
		latest, err := provider.GetLatestModVersion(dep.ModID)
		if err != nil {
			return false, "", fmt.Errorf("failed to fetch latest dependency %s: %w", dep.ModID, err)
		}
		if latest == nil {
			return false, "", fmt.Errorf("failed to fetch latest dependency %s: version not found", dep.ModID)
		}
		return resolvedDep.VersionID == latest.VersionID, fmt.Sprintf("latest (%s)", latest.VersionID), nil
	}

	return version.NewConstrainGroupFromString(constraint).Match(resolvedDep.VersionID), dep.VersionID, nil
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
	return fmt.Sprintf("%s@%s->%s@%s", current.ModID, current.VersionID, dep.ModID, dep.VersionID)
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

// getAllMatchingVersionIDs returns all versions that match the constraint, sorted from newest to oldest
func getAllMatchingVersionIDs(versionIDs []string, constraint string) []string {
	group := version.NewConstrainGroupFromString(constraint)
	matching := make([]string, 0)
	for _, versionID := range versionIDs {
		if group.Match(versionID) {
			matching = append(matching, versionID)
		}
	}

	// Sort in descending order (newest first)
	for i := 0; i < len(matching)-1; i++ {
		for j := i + 1; j < len(matching); j++ {
			if compareVersionID(matching[i], matching[j]) < 0 {
				matching[i], matching[j] = matching[j], matching[i]
			}
		}
	}

	return matching
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
