package discord

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertiesConversion(t *testing.T) {
	testMap := map[string]string{
		"profile_id":   "test-uuid-1234",
		"profile_name": "Town of Us",
		"room_code":    "ABCDEF",
		"ready":        "true",
	}

	props, cleanup := mapToProperties(testMap)
	defer cleanup()

	converted := propertiesToMap(props)
	assert.Equal(t, len(testMap), len(converted))
	for k, v := range testMap {
		assert.Equal(t, v, converted[k])
	}
}

func TestPropertiesConversionEmpty(t *testing.T) {
	props, cleanup := mapToProperties(nil)
	defer cleanup()

	converted := propertiesToMap(props)
	assert.NotNil(t, converted)
	assert.Empty(t, converted)
}

func TestLobbyInfoStructure(t *testing.T) {
	info := LobbyInfo{
		ID:         12345678,
		Secret:     "secret-hash-123",
		HostUserID: 99999,
		Metadata: map[string]string{
			"room_code": "XYZW",
		},
		Members: []LobbyMember{
			{
				UserID:      99999,
				DisplayName: "HostPlayer",
				IsHost:      true,
				IsReady:     true,
			},
			{
				UserID:      88888,
				DisplayName: "GuestPlayer",
				IsHost:      false,
				IsReady:     false,
			},
		},
		CreatedAt: time.Now(),
	}

	require.Equal(t, uint64(12345678), info.ID)
	require.Equal(t, "secret-hash-123", info.Secret)
	require.Equal(t, uint64(99999), info.HostUserID)
	require.Len(t, info.Members, 2)
	assert.True(t, info.Members[0].IsHost)
	assert.False(t, info.Members[1].IsHost)
}
