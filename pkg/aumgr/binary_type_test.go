package aumgr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectBinaryType(t *testing.T) {
	dir, err := GetAmongUsDir()
	if err != nil {
		t.Skipf("Among Us directory not found, skipping test: %v", err)
		return
	}

	exePath := filepath.Join(dir, "Among Us.exe")
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		t.Skipf("Among Us.exe not found at %s, skipping test", exePath)
		return
	}

	binaryType, err := DetectBinaryType(dir)
	if err != nil {
		t.Errorf("Failed to detect binary type: %v", err)
		return
	}

	switch binaryType {
	case BinaryType32Bit:
		t.Log("Detected 32-bit binary")
	case BinaryType64Bit:
		t.Log("Detected 64-bit binary")
	default:
		t.Errorf("Unexpected binary type: %s", binaryType)
	}
}

func TestDetectBinaryType_InvalidPath(t *testing.T) {
	_, err := DetectBinaryType("invalid/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for invalid path, but got nil")
	}
}
