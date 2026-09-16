package thunderstore

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDependency(t *testing.T) {
	tests := []struct {
		dep         string
		expectedMod string
		expectedVer string
	}{
		{"BepInEx-BepInExPack-5.4.2100", "BepInEx-BepInExPack", "5.4.2100"},
		{"Author-ModName-1.0.0", "Author-ModName", "1.0.0"},
		{"SingleString", "SingleString", ""},
		{"Author-Mod-SubName-2.3.4", "Author-Mod-SubName", "2.3.4"},
	}

	for _, tt := range tests {
		t.Run(tt.dep, func(t *testing.T) {
			modID, ver := ParseDependency(tt.dep)
			assert.Equal(t, tt.expectedMod, modID)
			assert.Equal(t, tt.expectedVer, ver)
		})
	}
}

func TestClient_SearchAndGet(t *testing.T) {
	samplePackages := []Package{
		{
			Name:        "BepInExPack",
			FullName:    "BepInEx-BepInExPack",
			Owner:       "BepInEx",
			DateCreated: time.Now(),
			DateUpdated: time.Now(),
			Categories:  []string{"Core"},
			Versions: []PackageVersion{
				{
					Name:          "BepInExPack",
					FullName:      "BepInEx-BepInExPack-5.4.2100",
					VersionNumber: "5.4.2100",
					Description:   "BepInEx pack for Unity Mono",
					Downloads:     1000,
					IsActive:      true,
				},
				{
					Name:          "BepInExPack",
					FullName:      "BepInEx-BepInExPack-5.4.2200",
					VersionNumber: "5.4.2200",
					Description:   "Latest BepInEx pack",
					Downloads:     500,
					IsActive:      true,
				},
			},
		},
		{
			Name:        "RepoQoL",
			FullName:    "Author-RepoQoL",
			Owner:       "Author",
			DateCreated: time.Now(),
			DateUpdated: time.Now(),
			Categories:  []string{"QoL"},
			Versions: []PackageVersion{
				{
					Name:          "RepoQoL",
					FullName:      "Author-RepoQoL-1.0.0",
					VersionNumber: "1.0.0",
					Description:   "Quality of life tweaks for REPO",
					Downloads:     250,
					IsActive:      true,
				},
			},
		},
	}

	tempDir, err := os.MkdirTemp("", "thunderstore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	client := NewClient(tempDir, nil)
	client.loadPackages(samplePackages)

	t.Run("GetPackage", func(t *testing.T) {
		pkg, ok := client.GetPackage("BepInEx-BepInExPack")
		assert.True(t, ok)
		assert.Equal(t, "BepInExPack", pkg.Name)
		assert.Equal(t, 2, len(pkg.Versions))
	})

	t.Run("GetPackageVersion", func(t *testing.T) {
		ver, ok := client.GetPackageVersion("BepInEx-BepInExPack", "5.4.2100")
		assert.True(t, ok)
		assert.Equal(t, "5.4.2100", ver.VersionNumber)

		latest, ok := client.GetLatestVersion("BepInEx-BepInExPack")
		assert.True(t, ok)
		assert.Equal(t, "5.4.2100", latest.VersionNumber)
	})

	t.Run("SearchPackages query", func(t *testing.T) {
		results := client.SearchPackages("qol", "", "downloads")
		require.Len(t, results, 1)
		assert.Equal(t, "Author-RepoQoL", results[0].FullName)
	})

	t.Run("SearchPackages category", func(t *testing.T) {
		results := client.SearchPackages("", "Core", "downloads")
		require.Len(t, results, 1)
		assert.Equal(t, "BepInEx-BepInExPack", results[0].FullName)
	})
}
