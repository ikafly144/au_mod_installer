package aumgr

import (
	"os"
	"testing"
)

func TestGetAmongUsDir(t *testing.T) {
	dir, err := GetAmongUsDir()
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("Among Us directory not found, skipping test")
			return
		}
		t.Errorf("Failed to get Among Us directory: %v", err)
		return
	}
	if dir == "" {
		t.Error("Expected Among Us directory to be found")
	}
	t.Logf("Among Us directory: %s", dir)
}

func TestDetectLauncherType(t *testing.T) {
	dir, err := GetAmongUsDir()
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("Among Us directory not found, skipping launcher type detection test")
			return
		}
		t.Errorf("Failed to get Among Us directory: %v", err)
		return
	}
	launcherType := DetectLauncherType(dir)
	if launcherType == LauncherUnknown {
		t.Error("Expected launcher type to be detected")
	}
	t.Logf("Detected launcher type: %v", launcherType)
}
