//go:build windows

package assetstools

import "testing"

func TestReadPlayerSettingsBundleVersion(t *testing.T) {
	gamePath, err := getAmongUsDir()
	if err != nil {
		t.Skipf("skipping: failed to locate Among Us directory: %v", err)
	}
	path := gamePath + "Among Us_Data\\globalgamemanagers"

	version, err := ReadPlayerSettingsBundleVersion(path)
	if err != nil {
		t.Fatalf("failed to read version: %v", err)
	}
	if version == "" {
		t.Fatal("version must not be empty")
	}
	t.Logf("version: %s", version)
}
