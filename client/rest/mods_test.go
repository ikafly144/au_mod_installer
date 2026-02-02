package rest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckForUpdates(t *testing.T) {
	// Create a temporary file for FileClient
	tempFile, err := os.CreateTemp("", "mods_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Sample data
	data := `[
		{
			"id": "mod-1",
			"name": "Mod 1",
			"latest_version": "v1.1.0",
			"versions": [
				{"id": "v1.0.0", "mod_id": "mod-1"},
				{"id": "v1.1.0", "mod_id": "mod-1"}
			]
		},
		{
			"id": "mod-2",
			"name": "Mod 2",
			"latest_version": "v2.0.0",
			"versions": [
				{"id": "v2.0.0", "mod_id": "mod-2"}
			]
		}
	]`
	_, err = tempFile.WriteString(data)
	require.NoError(t, err)
	tempFile.Close()

	client, err := NewFileClient(tempFile.Name())
	require.NoError(t, err)
	err = client.LoadData()
	require.NoError(t, err)

	installed := map[string]string{
		"mod-1": "v1.0.0", // Update available
		"mod-2": "v2.0.0", // Up to date
	}

	updates, err := client.CheckForUpdates(installed)
	require.NoError(t, err)

	assert.Len(t, updates, 1)
	assert.Contains(t, updates, "mod-1")
	assert.Equal(t, "v1.1.0", updates["mod-1"].ID)
	assert.NotContains(t, updates, "mod-2")
}
