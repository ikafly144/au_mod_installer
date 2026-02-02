package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientImpl_CheckForUpdates(t *testing.T) {
	// モックサーバーのセットアップ
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/mods/mod-1":
			// Mod 1 の詳細（最新バージョンは v1.1.0）
			mod := modmgr.Mod{ID: "mod-1", LatestVersion: "v1.1.0"}
			json.NewEncoder(w).Encode(mod)
		case r.URL.Path == "/mods/mod-1/versions/v1.1.0":
			// Mod 1 の最新バージョンの詳細
			version := modmgr.ModVersion{ID: "v1.1.0", ModID: "mod-1"}
			json.NewEncoder(w).Encode(version)
		case r.URL.Path == "/mods/mod-2":
			// Mod 2 の詳細（最新バージョンは v2.0.0）
			mod := modmgr.Mod{ID: "mod-2", LatestVersion: "v2.0.0"}
			json.NewEncoder(w).Encode(mod)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	installed := map[string]string{
		"mod-1": "v1.0.0", // アップデートあり
		"mod-2": "v2.0.0", // 最新
	}

	updates, err := client.CheckForUpdates(installed)
	require.NoError(t, err)

	assert.Len(t, updates, 1)
	assert.Contains(t, updates, "mod-1")
	assert.Equal(t, "v1.1.0", updates["mod-1"].ID)
	assert.NotContains(t, updates, "mod-2")
}
