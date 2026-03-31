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

func modVersion(modID, versionID string, deps ...model.ModVersionDependency) ModVersion {
	return ModVersion{
		ModVersionDetails: model.ModVersionDetails{
			VersionID:    versionID,
			ModID:        modID,
			Dependencies: deps,
		},
	}
}
