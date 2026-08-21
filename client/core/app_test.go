package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_ParseJoinGameURI(t *testing.T) {
	app := &App{}

	t.Run("valid join game URI without error_type", func(t *testing.T) {
		link, err := app.ParseJoinGameURI("mod-of-us://join_game/v1/test-session-123?server=http%3A%2F%2Flocalhost%3A8080")
		require.NoError(t, err)
		assert.Equal(t, "test-session-123", link.SessionID)
		assert.Equal(t, "http://localhost:8080", link.ServerBase)
		assert.Empty(t, link.ErrorType)
	})

	t.Run("valid join game URI with error_type", func(t *testing.T) {
		link, err := app.ParseJoinGameURI("mod-of-us://join_game/v1/test-session-123?error_type=invalid_session&server=http%3A%2F%2Flocalhost%3A8080")
		require.NoError(t, err)
		assert.Equal(t, "test-session-123", link.SessionID)
		assert.Equal(t, "http://localhost:8080", link.ServerBase)
		assert.Equal(t, "invalid_session", link.ErrorType)
	})

	t.Run("ignores legacy or arbitrary error query parameter", func(t *testing.T) {
		link, err := app.ParseJoinGameURI("mod-of-us://join_game/v1/test-session-123?error=arbitrary_text&server=http%3A%2F%2Flocalhost%3A8080")
		require.NoError(t, err)
		assert.Equal(t, "test-session-123", link.SessionID)
		assert.Equal(t, "http://localhost:8080", link.ServerBase)
		assert.Empty(t, link.ErrorType)
	})

	t.Run("invalid scheme", func(t *testing.T) {
		_, err := app.ParseJoinGameURI("http://join_game/v1/test-session-123?server=http%3A%2F%2Flocalhost%3A8080")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid join game URI")
	})

	t.Run("invalid host", func(t *testing.T) {
		_, err := app.ParseJoinGameURI("mod-of-us://other_host/v1/test-session-123?server=http%3A%2F%2Flocalhost%3A8080")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid join game URI")
	})

	t.Run("unsupported version", func(t *testing.T) {
		_, err := app.ParseJoinGameURI("mod-of-us://join_game/v2/test-session-123?server=http%3A%2F%2Flocalhost%3A8080")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported join game URI version")
	})

	t.Run("missing server", func(t *testing.T) {
		_, err := app.ParseJoinGameURI("mod-of-us://join_game/v1/test-session-123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "join game URI missing server")
	})

	t.Run("invalid server URL", func(t *testing.T) {
		_, err := app.ParseJoinGameURI("mod-of-us://join_game/v1/test-session-123?server=not-a-valid-url")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid join game URI server")
	})
}
