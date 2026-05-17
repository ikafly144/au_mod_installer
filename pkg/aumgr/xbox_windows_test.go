package aumgr

import "testing"

func TestXboxAppId(t *testing.T) {
	appId, err := GetXboxAppId()
	if err != nil {
		t.Fatalf("GetXboxAppId failed: %v", err)
	}
	if appId == "" {
		t.Fatal("GetXboxAppId returned empty AppId")
	}
	t.Logf("Xbox AppId: %s", appId)
}
