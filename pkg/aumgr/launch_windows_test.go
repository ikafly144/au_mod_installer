package aumgr

import "testing"

func TestIsSteamRunning(t *testing.T) {
	running, err := isSteamRunning()
	if err != nil {
		t.Errorf("Failed to check Steam process: %v", err)
		return
	}

	t.Logf("Steam running: %v", running)
}
