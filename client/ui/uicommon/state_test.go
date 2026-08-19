package uicommon

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestStateWindowVisibility(t *testing.T) {
	state := &State{}

	assert.False(t, state.IsWindowVisible())

	state.SetWindowVisible(true)
	assert.True(t, state.IsWindowVisible())

	state.SetWindowVisible(false)
	assert.False(t, state.IsWindowVisible())
}

func TestStateWindowVisibilityConcurrency(t *testing.T) {
	state := &State{}
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(2)
		go func(v bool) {
			defer wg.Done()
			state.SetWindowVisible(v)
		}(i%2 == 0)
		go func() {
			defer wg.Done()
			_ = state.IsWindowVisible()
		}()
	}
	wg.Wait()
}

func TestStateOnGameExitedListeners(t *testing.T) {
	state := &State{}

	var called1, called2, legacyCalled bool
	var calledProfile1, calledProfile2, legacyProfile uuid.UUID

	state.OnGameExited = func(id uuid.UUID) {
		legacyCalled = true
		legacyProfile = id
	}

	state.AddOnGameExitedListener(func(id uuid.UUID) {
		called1 = true
		calledProfile1 = id
	})

	state.AddOnGameExitedListener(func(id uuid.UUID) {
		called2 = true
		calledProfile2 = id
	})

	testID := uuid.New()

	if state.OnGameExited != nil {
		state.OnGameExited(testID)
	}
	state.gameExitedMu.Lock()
	listeners := make([]func(uuid.UUID), len(state.onGameExitedListeners))
	copy(listeners, state.onGameExitedListeners)
	state.gameExitedMu.Unlock()
	for _, l := range listeners {
		l(testID)
	}

	assert.True(t, legacyCalled)
	assert.Equal(t, testID, legacyProfile)
	assert.True(t, called1)
	assert.Equal(t, testID, calledProfile1)
	assert.True(t, called2)
	assert.Equal(t, testID, calledProfile2)
}
