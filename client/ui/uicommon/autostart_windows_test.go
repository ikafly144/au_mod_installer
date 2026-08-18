//go:build windows

package uicommon

import (
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestAutoStartRegistryToggle(t *testing.T) {
	// Remember original state
	originalState := IsAutoStartEnabled()

	// Disable auto start
	err := SetAutoStartEnabled(false, false)
	if err != nil {
		t.Fatalf("SetAutoStartEnabled(false, false) failed: %v", err)
	}
	if IsAutoStartEnabled() {
		t.Errorf("expected IsAutoStartEnabled() to be false after disabling")
	}

	// Enable auto start without startSilent
	err = SetAutoStartEnabled(true, false)
	if err != nil {
		t.Fatalf("SetAutoStartEnabled(true, false) failed: %v", err)
	}
	if !IsAutoStartEnabled() {
		t.Errorf("expected IsAutoStartEnabled() to be true after enabling")
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRegistryPath, registry.QUERY_VALUE)
	if err == nil {
		val, _, _ := key.GetStringValue(startupValueName)
		key.Close()
		if strings.Contains(val, "-silent") {
			t.Errorf("expected registry command to NOT contain -silent when startSilent is false, got %q", val)
		}
	}

	// Enable auto start with startSilent
	err = SetAutoStartEnabled(true, true)
	if err != nil {
		t.Fatalf("SetAutoStartEnabled(true, true) failed: %v", err)
	}
	if !IsAutoStartEnabled() {
		t.Errorf("expected IsAutoStartEnabled() to be true after enabling")
	}
	key, err = registry.OpenKey(registry.CURRENT_USER, startupRegistryPath, registry.QUERY_VALUE)
	if err == nil {
		val, _, _ := key.GetStringValue(startupValueName)
		key.Close()
		if !strings.Contains(val, "-silent") {
			t.Errorf("expected registry command to contain -silent when startSilent is true, got %q", val)
		}
	}

	// Restore original state
	err = SetAutoStartEnabled(originalState, false)
	if err != nil {
		t.Fatalf("failed to restore original autostart state: %v", err)
	}
	if IsAutoStartEnabled() != originalState {
		t.Errorf("expected restored autostart state to be %v, got %v", originalState, IsAutoStartEnabled())
	}
}
