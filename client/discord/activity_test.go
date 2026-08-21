package discord

import (
	"testing"

	discord "github.com/ikafly144/discord_social_sdk"
	"github.com/stretchr/testify/assert"
)

func TestIdleActivityToggle(t *testing.T) {
	client := discord.NewClient()
	service := NewDiscordService(client)

	assert.False(t, service.IsIdleActivityEnabled())

	idleCreated := false
	service.SetIdleActivity(func() *discord.Activity {
		idleCreated = true
		act := discord.NewActivity()
		act.SetName("Mod of Us")
		act.SetState("Idle")
		return act
	}, nil)

	// Since idleActivityEnabled is false by default, idle activity shouldn't be active yet
	act, hasActivity := service.CurrentActivity()
	assert.Nil(t, act)
	assert.False(t, hasActivity)

	// Enable idle activity (window shown)
	service.SetIdleActivityEnabled(true)
	assert.True(t, service.IsIdleActivityEnabled())
	assert.True(t, idleCreated)
	act, hasActivity = service.CurrentActivity()
	assert.NotNil(t, act)
	assert.True(t, hasActivity)
	state, ok := act.State()
	assert.True(t, ok)
	assert.Equal(t, "Idle", state)

	// Disable idle activity (window closed to tray)
	service.SetIdleActivityEnabled(false)
	assert.False(t, service.IsIdleActivityEnabled())
	act, hasActivity = service.CurrentActivity()
	assert.Nil(t, act)
	assert.False(t, hasActivity)

	// Start a game while window is closed (e.g. game launched)
	gameAct := discord.NewActivity()
	gameAct.SetName("Mod of Us")
	gameAct.SetState("In Game")
	service.SetActivity(gameAct, nil)

	act, hasActivity = service.CurrentActivity()
	assert.True(t, hasActivity)
	state, ok = act.State()
	assert.True(t, ok)
	assert.Equal(t, "In Game", state)

	// Disabling idle activity again while game is running should NOT clear the game activity
	service.SetIdleActivityEnabled(false)
	act, hasActivity = service.CurrentActivity()
	assert.True(t, hasActivity)
	state, ok = act.State()
	assert.True(t, ok)
	assert.Equal(t, "In Game", state)

	// Game exits while window is still closed -> should clear activity completely (not revert to idle)
	service.ClearActivity()
	act, hasActivity = service.CurrentActivity()
	assert.Nil(t, act)
	assert.False(t, hasActivity)

	// Window is reopened -> idle activity should be restored
	service.SetIdleActivityEnabled(true)
	act, hasActivity = service.CurrentActivity()
	assert.NotNil(t, act)
	assert.True(t, hasActivity)
	state, ok = act.State()
	assert.True(t, ok)
	assert.Equal(t, "Idle", state)
}

func TestSentJoinRequestTracking(t *testing.T) {
	client := discord.NewClient()
	service := NewDiscordService(client)

	userID := uint64(123456789)

	// Initially, no sent join request
	assert.False(t, service.ConsumeSentJoinRequest(userID))

	// Record sent join request
	service.RecordSentJoinRequest(userID)

	// First consume should succeed and remove it
	assert.True(t, service.ConsumeSentJoinRequest(userID))

	// Second consume should fail because it was already consumed
	assert.False(t, service.ConsumeSentJoinRequest(userID))

	// Record and then explicitly remove
	service.RecordSentJoinRequest(userID)
	service.RemoveSentJoinRequest(userID)
	assert.False(t, service.ConsumeSentJoinRequest(userID))
}
