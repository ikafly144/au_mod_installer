package profile

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		ID:    versionID,
		ModID: modID,
	})

	err = manager.Add(p)
	require.NoError(t, err)

	// Verify it was saved and can be loaded
	loadedProfile, ok := manager.Get(profileID)
	assert.True(t, ok)
	assert.Equal(t, "Test Profile", loadedProfile.Name)
	assert.Equal(t, versionID, loadedProfile.ModVersions[modID].ID)

	// Verify persistence
	newManager, err := NewManager(tempDir)
	require.NoError(t, err)
	persistedProfile, ok := newManager.Get(profileID)
	assert.True(t, ok)
	assert.Equal(t, versionID, persistedProfile.ModVersions[modID].ID)
}

func TestProfile_VersionTracking(t *testing.T) {
	p := Profile{
		ID:   uuid.New(),
		Name: "Tracking Profile",
	}

	modID := "example-mod"
	v1 := modmgr.ModVersion{ID: "v1", ModID: modID}
	v2 := modmgr.ModVersion{ID: "v2", ModID: modID}

	p.AddModVersion(v1)
	assert.Equal(t, "v1", p.ModVersions[modID].ID)

	// Updating the version for the same mod
	p.AddModVersion(v2)
	assert.Equal(t, "v2", p.ModVersions[modID].ID)
}
