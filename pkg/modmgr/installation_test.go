package modmgr

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallationInfo_SaveAndLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "install_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	gameRoot, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer gameRoot.Close()

	modVersion := ModVersion{
		ID:    "v1.2.3",
		ModID: "test-mod",
	}

	installation := &ModInstallation{
		FileVersion: currentFileVersion,
		InstalledMods: []InstalledVersionInfo{
			{
				ModVersion: modVersion,
				Paths:      []string{"test.dll"},
			},
		},
		InstalledGameVersion: "2024.6.18",
		Status:               InstallStatusCompatible,
	}

	err = SaveInstallationInfo(gameRoot, installation)
	require.NoError(t, err)

	loaded, err := LoadInstallationInfo(gameRoot)
	require.NoError(t, err)

	assert.Equal(t, installation.FileVersion, loaded.FileVersion)
	assert.Equal(t, installation.InstalledGameVersion, loaded.InstalledGameVersion)
	require.Len(t, loaded.InstalledMods, 1)
	assert.Equal(t, "v1.2.3", loaded.InstalledMods[0].ID)
	assert.Equal(t, "test-mod", loaded.InstalledMods[0].ModID)
}
