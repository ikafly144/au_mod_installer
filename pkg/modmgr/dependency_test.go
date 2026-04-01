package modmgr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ikafly144/au_mod_installer/common/rest/model"
)

type mockVersionProvider struct {
	versions map[string]map[string]ModVersion
	ids      map[string][]string
	latest   map[string]string
}

func (m *mockVersionProvider) GetModVersion(modID string, versionID string) (*ModVersion, error) {
	modVersions, ok := m.versions[modID]
	if !ok {
		return nil, nil
	}
	v, ok := modVersions[versionID]
	if !ok {
		return nil, nil
	}
	return &v, nil
}

func (m *mockVersionProvider) GetLatestModVersion(modID string) (*ModVersion, error) {
	latestID, ok := m.latest[modID]
	if !ok {
		return nil, nil
	}
	return m.GetModVersion(modID, latestID)
}

func (m *mockVersionProvider) GetModVersionIDs(modID string, limit int, after string) ([]string, error) {
	ids, ok := m.ids[modID]
	if !ok {
		return nil, fmt.Errorf("mod %s not found", modID)
	}
	return ids, nil
}

func TestResolveDependencies_ExactVersionConstraint(t *testing.T) {
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": modVersion("a", "v1.0.0", model.ModVersionDependency{
				ModID:          "b",
				VersionID:      "v1.1.0",
				DependencyType: model.DependencyTypeRequired,
			})},
			"b": {"v1.1.0": modVersion("b", "v1.1.0")},
		},
		ids: map[string][]string{"b": {"v1.1.0"}},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v1.1.0",
		},
	}

	resolved, err := ResolveDependencies([]ModVersion{modVersion("a", "v1.0.0", model.ModVersionDependency{
		ModID:          "b",
		VersionID:      "v1.1.0",
		DependencyType: model.DependencyTypeRequired,
	})}, provider)
	require.NoError(t, err)
	require.Equal(t, "v1.1.0", resolved["b"].VersionID)
}

func TestResolveDependencies_ConstraintSelectsHighestMatch(t *testing.T) {
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": modVersion("a", "v1.0.0", model.ModVersionDependency{
				ModID:          "b",
				VersionID:      ">=v1.0.0, <v2.0.0",
				DependencyType: model.DependencyTypeRequired,
			})},
			"b": {
				"v1.0.0": modVersion("b", "v1.0.0"),
				"v1.5.0": modVersion("b", "v1.5.0"),
				"v2.0.0": modVersion("b", "v2.0.0"),
			},
		},
		ids: map[string][]string{"b": {"v1.0.0", "v2.0.0", "v1.5.0"}},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v2.0.0",
		},
	}

	resolved, err := ResolveDependencies([]ModVersion{modVersion("a", "v1.0.0", model.ModVersionDependency{
		ModID:          "b",
		VersionID:      ">=v1.0.0, <v2.0.0",
		DependencyType: model.DependencyTypeRequired,
	})}, provider)
	require.NoError(t, err)
	require.Equal(t, "v1.5.0", resolved["b"].VersionID)
}

func TestResolveDependencies_ConstraintConflictOnResolved(t *testing.T) {
	initial := []ModVersion{
		modVersion("b", "v1.0.0"),
		modVersion("a", "v1.0.0", model.ModVersionDependency{
			ModID:          "b",
			VersionID:      ">=v1.1.0",
			DependencyType: model.DependencyTypeRequired,
		}),
	}
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": initial[1]},
			"b": {"v1.0.0": initial[0], "v1.1.0": modVersion("b", "v1.1.0")},
		},
		ids: map[string][]string{"b": {"v1.0.0", "v1.1.0"}},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v1.1.0",
		},
	}

	_, err := ResolveDependencies(initial, provider)
	require.Error(t, err)
	require.Contains(t, err.Error(), "version conflict for mod b")
}

func TestResolveDependencies_LatestConstraintOnResolved(t *testing.T) {
	initial := []ModVersion{
		modVersion("b", "v1.0.0"),
		modVersion("a", "v1.0.0", model.ModVersionDependency{
			ModID:          "b",
			VersionID:      "latest",
			DependencyType: model.DependencyTypeRequired,
		}),
	}
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": initial[1]},
			"b": {"v1.0.0": initial[0], "v2.0.0": modVersion("b", "v2.0.0")},
		},
		ids: map[string][]string{"b": {"v1.0.0", "v2.0.0"}},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v2.0.0",
		},
	}

	_, err := ResolveDependencies(initial, provider)
	require.Error(t, err)
	require.Contains(t, err.Error(), "required latest")
}

func TestResolveDependencies_OptionalNotAutoInstalled(t *testing.T) {
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": modVersion("a", "v1.0.0", model.ModVersionDependency{
				ModID:          "b",
				VersionID:      "any",
				DependencyType: model.DependencyTypeOptional,
			})},
			"b": {"v1.2.0": modVersion("b", "v1.2.0")},
		},
		ids: map[string][]string{"b": {"v1.2.0"}},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v1.2.0",
		},
	}

	resolved, err := ResolveDependencies([]ModVersion{modVersion("a", "v1.0.0", model.ModVersionDependency{
		ModID:          "b",
		VersionID:      "any",
		DependencyType: model.DependencyTypeOptional,
	})}, provider)
	require.NoError(t, err)
	require.Contains(t, resolved, "a")
	require.NotContains(t, resolved, "b")
}

func TestResolveDependencies_OptionalConstraintCheckedWhenPresent(t *testing.T) {
	initial := []ModVersion{
		modVersion("b", "v1.0.0"),
		modVersion("a", "v1.0.0", model.ModVersionDependency{
			ModID:          "b",
			VersionID:      ">=v1.1.0",
			DependencyType: model.DependencyTypeOptional,
		}),
	}
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": initial[1]},
			"b": {"v1.0.0": initial[0], "v1.1.0": modVersion("b", "v1.1.0")},
		},
		ids: map[string][]string{"b": {"v1.0.0", "v1.1.0"}},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v1.1.0",
		},
	}

	_, err := ResolveDependencies(initial, provider)
	require.Error(t, err)
	require.Contains(t, err.Error(), "optional dependency version conflict for mod b")
}

func TestResolveDependencies_ConflictErrorsWhenPresent(t *testing.T) {
	initial := []ModVersion{
		modVersion("b", "v1.2.0"),
		modVersion("a", "v1.0.0", model.ModVersionDependency{
			ModID:          "b",
			VersionID:      ">=v1.0.0",
			DependencyType: model.DependencyTypeConflict,
		}),
	}
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": initial[1]},
			"b": {"v1.2.0": initial[0]},
		},
		ids: map[string][]string{"b": {"v1.2.0"}},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v1.2.0",
		},
	}

	_, err := ResolveDependencies(initial, provider)
	require.Error(t, err)
	require.Contains(t, err.Error(), "dependency conflict for mod b")
}

func TestResolveDependencies_ConflictOrderIndependent(t *testing.T) {
	a := modVersion("a", "v1.0.0", model.ModVersionDependency{
		ModID:          "b",
		VersionID:      "any",
		DependencyType: model.DependencyTypeConflict,
	})
	b := modVersion("b", "v1.2.0")
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": a},
			"b": {"v1.2.0": b},
		},
		ids: map[string][]string{"b": {"v1.2.0"}},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v1.2.0",
		},
	}

	_, err1 := ResolveDependencies([]ModVersion{a, b}, provider)
	require.Error(t, err1)
	require.Contains(t, err1.Error(), "dependency conflict for mod b")

	_, err2 := ResolveDependencies([]ModVersion{b, a}, provider)
	require.Error(t, err2)
	require.Contains(t, err2.Error(), "dependency conflict for mod b")
}

func TestResolveDependencies_ConflictIgnoredWhenMissing(t *testing.T) {
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": modVersion("a", "v1.0.0", model.ModVersionDependency{
				ModID:          "b",
				VersionID:      "any",
				DependencyType: model.DependencyTypeConflict,
			})},
		},
		latest: map[string]string{
			"a": "v1.0.0",
		},
	}

	resolved, err := ResolveDependencies([]ModVersion{modVersion("a", "v1.0.0", model.ModVersionDependency{
		ModID:          "b",
		VersionID:      "any",
		DependencyType: model.DependencyTypeConflict,
	})}, provider)
	require.NoError(t, err)
	require.Contains(t, resolved, "a")
	require.NotContains(t, resolved, "b")
}

func TestResolveDependencies_EmbeddedIgnoredWhenMissing(t *testing.T) {
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": modVersion("a", "v1.0.0", model.ModVersionDependency{
				ModID:          "b",
				VersionID:      "any",
				DependencyType: model.DependencyTypeEmbedded,
			})},
		},
		latest: map[string]string{
			"a": "v1.0.0",
		},
	}

	resolved, err := ResolveDependencies([]ModVersion{modVersion("a", "v1.0.0", model.ModVersionDependency{
		ModID:          "b",
		VersionID:      "any",
		DependencyType: model.DependencyTypeEmbedded,
	})}, provider)
	require.NoError(t, err)
	require.Contains(t, resolved, "a")
	require.NotContains(t, resolved, "b")
}

func TestResolveDependencies_EmbeddedCanCoexistWhenPresent(t *testing.T) {
	initial := []ModVersion{
		modVersion("b", "v1.0.0"),
		modVersion("a", "v1.0.0", model.ModVersionDependency{
			ModID:          "b",
			VersionID:      "any",
			DependencyType: model.DependencyTypeEmbedded,
		}),
	}
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": initial[1]},
			"b": {"v1.0.0": initial[0]},
		},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v1.0.0",
		},
	}

	resolved, err := ResolveDependencies(initial, provider)
	require.NoError(t, err)
	require.Contains(t, resolved, "a")
	require.Contains(t, resolved, "b")
}

func TestResolveDependencies_RequiredCanBeSatisfiedByEmbedded(t *testing.T) {
	initial := []ModVersion{
		modVersion("b", "v1.0.0", model.ModVersionDependency{
			ModID:          "a",
			VersionID:      "any",
			DependencyType: model.DependencyTypeRequired,
		}),
		modVersion("c", "v1.0.0", model.ModVersionDependency{
			ModID:          "a",
			VersionID:      "any",
			DependencyType: model.DependencyTypeEmbedded,
		}),
	}
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": modVersion("a", "v1.0.0")},
			"b": {"v1.0.0": initial[0]},
			"c": {"v1.0.0": initial[1]},
		},
		ids: map[string][]string{"a": {"v1.0.0"}},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v1.0.0",
			"c": "v1.0.0",
		},
	}

	resolved, err := ResolveDependencies(initial, provider)
	require.NoError(t, err)
	require.Contains(t, resolved, "b")
	require.Contains(t, resolved, "c")
	require.NotContains(t, resolved, "a")
}

func TestResolveDependencies_BacktrackToOlderVersion(t *testing.T) {
	// Scenario: Mod A requires B >=1.0.0, Mod C requires B <2.0.0
	// Available versions of B: 1.0.0, 1.5.0, 2.0.0, 2.5.0
	// Expected: Should select B 1.5.0 (not latest 2.5.0) to satisfy both constraints
	initial := []ModVersion{
		modVersion("a", "v1.0.0", model.ModVersionDependency{
			ModID:          "b",
			VersionID:      ">=v1.0.0",
			DependencyType: model.DependencyTypeRequired,
		}),
		modVersion("c", "v1.0.0", model.ModVersionDependency{
			ModID:          "b",
			VersionID:      "<v2.0.0",
			DependencyType: model.DependencyTypeRequired,
		}),
	}
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": initial[0]},
			"b": {
				"v1.0.0": modVersion("b", "v1.0.0"),
				"v1.5.0": modVersion("b", "v1.5.0"),
				"v2.0.0": modVersion("b", "v2.0.0"),
				"v2.5.0": modVersion("b", "v2.5.0"),
			},
			"c": {"v1.0.0": initial[1]},
		},
		ids: map[string][]string{"b": {"v1.0.0", "v2.5.0", "v2.0.0", "v1.5.0"}},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v2.5.0",
			"c": "v1.0.0",
		},
	}

	resolved, err := ResolveDependencies(initial, provider)
	require.NoError(t, err)
	require.Equal(t, "v1.5.0", resolved["b"].VersionID, "Should select v1.5.0 to satisfy both constraints")
}

func TestResolveDependencies_BacktrackWithTransitiveDependencies(t *testing.T) {
	// Scenario:
	// A requires B >=1.0.0
	// B v2.0.0 requires D >=2.0.0
	// B v1.0.0 requires D >=1.0.0
	// C requires D <2.0.0
	// Expected: Should select B v1.0.0 and D v1.5.0 to satisfy all constraints
	bV2 := modVersion("b", "v2.0.0", model.ModVersionDependency{
		ModID:          "d",
		VersionID:      ">=v2.0.0",
		DependencyType: model.DependencyTypeRequired,
	})
	bV1 := modVersion("b", "v1.0.0", model.ModVersionDependency{
		ModID:          "d",
		VersionID:      ">=v1.0.0",
		DependencyType: model.DependencyTypeRequired,
	})

	initial := []ModVersion{
		modVersion("a", "v1.0.0", model.ModVersionDependency{
			ModID:          "b",
			VersionID:      ">=v1.0.0",
			DependencyType: model.DependencyTypeRequired,
		}),
		modVersion("c", "v1.0.0", model.ModVersionDependency{
			ModID:          "d",
			VersionID:      "<v2.0.0",
			DependencyType: model.DependencyTypeRequired,
		}),
	}

	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": initial[0]},
			"b": {
				"v1.0.0": bV1,
				"v2.0.0": bV2,
			},
			"c": {"v1.0.0": initial[1]},
			"d": {
				"v1.0.0": modVersion("d", "v1.0.0"),
				"v1.5.0": modVersion("d", "v1.5.0"),
				"v2.0.0": modVersion("d", "v2.0.0"),
			},
		},
		ids: map[string][]string{
			"b": {"v2.0.0", "v1.0.0"},
			"d": {"v2.0.0", "v1.5.0", "v1.0.0"},
		},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v2.0.0",
			"c": "v1.0.0",
			"d": "v2.0.0",
		},
	}

	resolved, err := ResolveDependencies(initial, provider)
	require.NoError(t, err)
	require.Equal(t, "v1.0.0", resolved["b"].VersionID, "Should backtrack to B v1.0.0")
	require.Contains(t, []string{"v1.0.0", "v1.5.0"}, resolved["d"].VersionID, "Should select D < v2.0.0")
}

func TestResolveDependencies_PreferNewerWhenNoConflict(t *testing.T) {
	// Scenario: A requires B >=1.0.0, no other constraints
	// Available versions of B: 1.0.0, 1.5.0, 2.0.0
	// Expected: Should select latest matching version (2.0.0)
	initial := []ModVersion{
		modVersion("a", "v1.0.0", model.ModVersionDependency{
			ModID:          "b",
			VersionID:      ">=v1.0.0",
			DependencyType: model.DependencyTypeRequired,
		}),
	}
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {"v1.0.0": initial[0]},
			"b": {
				"v1.0.0": modVersion("b", "v1.0.0"),
				"v1.5.0": modVersion("b", "v1.5.0"),
				"v2.0.0": modVersion("b", "v2.0.0"),
			},
		},
		ids: map[string][]string{"b": {"v1.0.0", "v2.0.0", "v1.5.0"}},
		latest: map[string]string{
			"a": "v1.0.0",
			"b": "v2.0.0",
		},
	}

	resolved, err := ResolveDependencies(initial, provider)
	require.NoError(t, err)
	require.Equal(t, "v2.0.0", resolved["b"].VersionID, "Should prefer newest version when no conflicts")
}

func TestResolveDependencies_MergeConstraintsFromMultipleMods(t *testing.T) {
	// Scenario:
	// MOD B requires A >=1.0.0
	// MOD C requires A >=1.5.0, <2.0.0
	// Available versions of A: 1.0.0, 1.5.0, 1.8.0, 2.0.0, 2.5.0
	// Expected: Should select A 1.8.0 (satisfies both >=1.0.0 AND >=1.5.0, <2.0.0)
	initial := []ModVersion{
		modVersion("b", "v1.0.0", model.ModVersionDependency{
			ModID:          "a",
			VersionID:      ">=v1.0.0",
			DependencyType: model.DependencyTypeRequired,
		}),
		modVersion("c", "v1.0.0", model.ModVersionDependency{
			ModID:          "a",
			VersionID:      ">=v1.5.0, <v2.0.0",
			DependencyType: model.DependencyTypeRequired,
		}),
	}
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {
				"v1.0.0": modVersion("a", "v1.0.0"),
				"v1.5.0": modVersion("a", "v1.5.0"),
				"v1.8.0": modVersion("a", "v1.8.0"),
				"v2.0.0": modVersion("a", "v2.0.0"),
				"v2.5.0": modVersion("a", "v2.5.0"),
			},
			"b": {"v1.0.0": initial[0]},
			"c": {"v1.0.0": initial[1]},
		},
		ids: map[string][]string{"a": {"v2.5.0", "v2.0.0", "v1.8.0", "v1.5.0", "v1.0.0"}},
		latest: map[string]string{
			"a": "v2.5.0",
			"b": "v1.0.0",
			"c": "v1.0.0",
		},
	}

	resolved, err := ResolveDependencies(initial, provider)
	require.NoError(t, err)
	require.Equal(t, "v1.8.0", resolved["a"].VersionID, "Should select v1.8.0 satisfying merged constraints >=1.5.0, <2.0.0")
}

func TestResolveDependencies_MergeConstraintsWithExactVersion(t *testing.T) {
	// Scenario:
	// MOD B requires A >=1.5.0
	// MOD C requires A v1.8.0 (exact)
	// Expected: Should select A v1.8.0 (the exact constraint is narrower)
	initial := []ModVersion{
		modVersion("b", "v1.0.0", model.ModVersionDependency{
			ModID:          "a",
			VersionID:      ">=v1.5.0",
			DependencyType: model.DependencyTypeRequired,
		}),
		modVersion("c", "v1.0.0", model.ModVersionDependency{
			ModID:          "a",
			VersionID:      "v1.8.0",
			DependencyType: model.DependencyTypeRequired,
		}),
	}
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {
				"v1.5.0": modVersion("a", "v1.5.0"),
				"v1.8.0": modVersion("a", "v1.8.0"),
				"v2.0.0": modVersion("a", "v2.0.0"),
			},
			"b": {"v1.0.0": initial[0]},
			"c": {"v1.0.0": initial[1]},
		},
		ids: map[string][]string{"a": {"v2.0.0", "v1.8.0", "v1.5.0"}},
		latest: map[string]string{
			"a": "v2.0.0",
			"b": "v1.0.0",
			"c": "v1.0.0",
		},
	}

	resolved, err := ResolveDependencies(initial, provider)
	require.NoError(t, err)
	require.Equal(t, "v1.8.0", resolved["a"].VersionID, "Should select exact version v1.8.0")
}

func TestResolveDependencies_MergeConstraintsNoValidVersion(t *testing.T) {
	// Scenario:
	// MOD B requires A >=2.0.0
	// MOD C requires A <2.0.0
	// Expected: Should fail - no version satisfies both constraints
	initial := []ModVersion{
		modVersion("b", "v1.0.0", model.ModVersionDependency{
			ModID:          "a",
			VersionID:      ">=v2.0.0",
			DependencyType: model.DependencyTypeRequired,
		}),
		modVersion("c", "v1.0.0", model.ModVersionDependency{
			ModID:          "a",
			VersionID:      "<v2.0.0",
			DependencyType: model.DependencyTypeRequired,
		}),
	}
	provider := &mockVersionProvider{
		versions: map[string]map[string]ModVersion{
			"a": {
				"v1.5.0": modVersion("a", "v1.5.0"),
				"v2.0.0": modVersion("a", "v2.0.0"),
				"v2.5.0": modVersion("a", "v2.5.0"),
			},
			"b": {"v1.0.0": initial[0]},
			"c": {"v1.0.0": initial[1]},
		},
		ids: map[string][]string{"a": {"v2.5.0", "v2.0.0", "v1.5.0"}},
		latest: map[string]string{
			"a": "v2.5.0",
			"b": "v1.0.0",
			"c": "v1.0.0",
		},
	}

	_, err := ResolveDependencies(initial, provider)
	require.Error(t, err)
	require.Contains(t, err.Error(), "satisfying all constraints")
}

func modVersion(modID, versionID string, deps ...model.ModVersionDependency) ModVersion {
	return ModVersion{
		ModVersionDetails: model.ModVersionDetails{
			VersionID:    versionID,
			ModID:        modID,
			Dependencies: deps,
		},
	}
}
