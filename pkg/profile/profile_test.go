package profile

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ikafly144/au_mod_installer/common/rest/model"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

func TestProfileManager_AddAndGet(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "profile_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	manager, err := NewManager(tempDir)
	require.NoError(t, err)

	profileID := uuid.New()
	modID := "test-mod"
	versionID := "v1.0.0"

	p := Profile{
		ID:        profileID,
		Name:      "Test Profile",
		UpdatedAt: time.Now(),
	}
	p.AddModVersion(modmgr.ModVersion{
		VersionID: versionID,
		ModID:     modID,
	})

	err = manager.Add(p)
	require.NoError(t, err)

	// Verify it was saved and can be loaded
	loadedProfile, ok := manager.Get(profileID)
	assert.True(t, ok)
	assert.Equal(t, "Test Profile", loadedProfile.Name)
	assert.Equal(t, versionID, loadedProfile.ModVersions[modID].VersionID)

	// Verify persistence
	newManager, err := NewManager(tempDir)
	require.NoError(t, err)
	persistedProfile, ok := newManager.Get(profileID)
	assert.True(t, ok)
	assert.Equal(t, versionID, persistedProfile.ModVersions[modID].VersionID)
}

func TestProfile_VersionTracking(t *testing.T) {
	p := Profile{
		ID:   uuid.New(),
		Name: "Tracking Profile",
	}

	modID := "example-mod"
	v1 := modmgr.ModVersion{
		VersionID: "v1",
		ModID:     modID,
	}
	v2 := modmgr.ModVersion{
		VersionID: "v2",
		ModID:     modID,
	}

	p.AddModVersion(v1)
	assert.Equal(t, "v1", p.ModVersions[modID].VersionID)

	// Updating the version for the same mod
	p.AddModVersion(v2)
	assert.Equal(t, "v2", p.ModVersions[modID].VersionID)
}

func TestProfileManager_IconFileCRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "profile_icon_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	manager, err := NewManager(tempDir)
	require.NoError(t, err)

	id := uuid.New()
	icon := []byte("png-bytes")

	err = manager.SaveIconPNG(id, icon)
	require.NoError(t, err)

	loaded, err := manager.LoadIconPNG(id)
	require.NoError(t, err)
	assert.Equal(t, icon, loaded)

	err = manager.RemoveIcon(id)
	require.NoError(t, err)

	loaded, err = manager.LoadIconPNG(id)
	require.NoError(t, err)
	assert.Nil(t, loaded)
}

func TestProfile_MatchesShared(t *testing.T) {
	id := uuid.New()
	p := &Profile{
		ID: id,
		ModVersions: map[string]modmgr.ModVersion{
			"mod-a": {ModVersionDetails: model.ModVersionDetails{ModID: "mod-a", VersionID: "1.0.0"}},
			"mod-b": {ModVersionDetails: model.ModVersionDetails{ModID: "mod-b", VersionID: "2.0.0"}},
		},
	}

	t.Run("matching shared profile", func(t *testing.T) {
		shared := SharedProfile{
			ID: id,
			ModVersions: map[string]string{
				"mod-a": "1.0.0",
				"mod-b": "2.0.0",
			},
		}
		assert.True(t, p.MatchesShared(shared))
		assert.True(t, p.MatchesSharedModVersions(shared))
	})

	t.Run("different profile ID", func(t *testing.T) {
		shared := SharedProfile{
			ID: uuid.New(),
			ModVersions: map[string]string{
				"mod-a": "1.0.0",
				"mod-b": "2.0.0",
			},
		}
		assert.False(t, p.MatchesShared(shared))
		assert.True(t, p.MatchesSharedModVersions(shared))
	})

	t.Run("different version for existing mod", func(t *testing.T) {
		shared := SharedProfile{
			ID: id,
			ModVersions: map[string]string{
				"mod-a": "1.0.0",
				"mod-b": "2.1.0",
			},
		}
		assert.False(t, p.MatchesShared(shared))
		assert.False(t, p.MatchesSharedModVersions(shared))
	})

	t.Run("missing a mod in shared", func(t *testing.T) {
		shared := SharedProfile{
			ID: id,
			ModVersions: map[string]string{
				"mod-a": "1.0.0",
			},
		}
		assert.False(t, p.MatchesShared(shared))
		assert.False(t, p.MatchesSharedModVersions(shared))
	})

	t.Run("extra mod in shared", func(t *testing.T) {
		shared := SharedProfile{
			ID: id,
			ModVersions: map[string]string{
				"mod-a": "1.0.0",
				"mod-b": "2.0.0",
				"mod-c": "3.0.0",
			},
		}
		assert.False(t, p.MatchesShared(shared))
		assert.False(t, p.MatchesSharedModVersions(shared))
	})

	t.Run("both empty mods", func(t *testing.T) {
		emptyP := &Profile{ID: id}
		shared := SharedProfile{ID: id}
		assert.True(t, emptyP.MatchesShared(shared))
		assert.True(t, emptyP.MatchesSharedModVersions(shared))
	})
}
