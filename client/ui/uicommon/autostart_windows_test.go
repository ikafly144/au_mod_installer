//go:build windows

package uicommon

import (
	"testing"
)

func TestAutoStartRegistryToggle(t *testing.T) {
	// Remember original state
	originalState := IsAutoStartEnabled()

	// Disable auto start
	err := SetAutoStartEnabled(false)
	if err != nil {
		t.Fatalf("SetAutoStartEnabled(false) failed: %v", err)
	}
	if IsAutoStartEnabled() {
		t.Errorf("expected IsAutoStartEnabled() to be false after disabling")
	}

	// Enable auto start
	err = SetAutoStartEnabled(true)
	if err != nil {
		t.Fatalf("SetAutoStartEnabled(true) failed: %v", err)
	}
	if !IsAutoStartEnabled() {
		t.Errorf("expected IsAutoStartEnabled() to be true after enabling")
	}

	// Restore original state
	err = SetAutoStartEnabled(originalState)
	if err != nil {
		t.Fatalf("failed to restore original autostart state: %v", err)
	}
	if IsAutoStartEnabled() != originalState {
		t.Errorf("expected restored autostart state to be %v, got %v", originalState, IsAutoStartEnabled())
	}
}
