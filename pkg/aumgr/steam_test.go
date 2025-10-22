package aumgr

import "testing"

const steamGameDir = `C:\Program Files (x86)\Steam\steamapps\common\Among Us\`

func TestGetSteamManifest(t *testing.T) {
	manifest, err := getSteamManifest(steamGameDir)
	if err != nil || manifest == nil {
		t.Errorf("Failed to get Steam manifest: %v", err)
		return
	}
	t.Logf("Steam manifest: %#v", manifest)
}
