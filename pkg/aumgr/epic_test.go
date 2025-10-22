package aumgr

import "testing"

func TestGetEpicManifest(t *testing.T) {
	manifest, err := getEpicManifest()
	if err != nil || manifest == nil {
		t.Errorf("Failed to get Epic Games manifest: %v", err)
		return
	}
	t.Logf("Epic Games manifest: %#v", manifest)
}
