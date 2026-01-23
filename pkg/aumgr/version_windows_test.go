//go:build windows

package aumgr

import (
	"path/filepath"
	"testing"
)

func TestReadVersionFile(t *testing.T) {
	dir, err := GetAmongUsDir()
	if err != nil {
		t.Skipf("Skipping due to failed to get Among Us directory: %s", err.Error())
	}
	t.Logf("Among Us directory: %s", dir)
	version, err := readVersionFile(filepath.Join(dir, "Among Us_Data", "globalgamemanagers"))
	if err != nil {
		t.Fatalf("Failed to read version file: %s", err.Error())
	}
	t.Logf("Among Us version: %s", version)
}
