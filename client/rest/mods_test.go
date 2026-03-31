package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	restcommon "github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/common/rest/model"
)

func TestClientImpl_CheckForUpdates(t *testing.T) {
	// モックサーバーのセットアップ
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Received request: %s %s", r.Method, r.URL.Path)
		switch r.URL.Path {
		case "/mod/mod-1":
			// Mod 1 の詳細（最新バージョンは v1.1.0）
			mod := model.ModDetails{ID: "mod-1", LatestVersionID: "v1.1.0"}
			if err := json.NewEncoder(w).Encode(mod); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		case "/mod/mod-1/version/v1.1.0":
			// Mod 1 の最新バージョンの詳細
			version := model.ModVersionDetails{ID: "v1.1.0", ModID: "mod-1"}
			if err := json.NewEncoder(w).Encode(version); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		case "/mod/mod-2":
			// Mod 2 の詳細（最新バージョンは v2.0.0）
			mod := model.ModDetails{ID: "mod-2", LatestVersionID: "v2.0.0"}
			if err := json.NewEncoder(w).Encode(mod); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		case "/mod/mod-2/version/v2.0.0":
			// Mod 2 の最新バージョンの詳細
			version := model.ModVersionDetails{ID: "v2.0.0", ModID: "mod-2"}
			if err := json.NewEncoder(w).Encode(version); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		default:
			t.Errorf("Unexpected request path: %s", r.URL.Path)
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

func TestClientImpl_ShareGame_UsesMultipartFormData(t *testing.T) {
	expectedAupack := []byte("test-aupack-bytes")
	expectedRoom := restcommon.RoomInfo{
		LobbyCode:  "ABCD",
		ServerIP:   "127.0.0.1",
		ServerPort: 22023,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/share_game", r.URL.Path)
		require.True(t, strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;"))

		require.NoError(t, r.ParseMultipartForm(1<<20))
		file, _, err := r.FormFile("aupack")
		require.NoError(t, err)
		defer file.Close()

		gotAupack, err := io.ReadAll(file)
		require.NoError(t, err)
		assert.Equal(t, expectedAupack, gotAupack)
		assert.Equal(t, expectedRoom.LobbyCode, r.FormValue("lobby_code"))
		assert.Equal(t, expectedRoom.ServerIP, r.FormValue("server_ip"))
		assert.Equal(t, "22023", r.FormValue("server_port"))

		require.NoError(t, json.NewEncoder(w).Encode(restcommon.ShareGameResponse{
			URL:       "/join_game?session_id=s1",
			SessionID: "s1",
			HostKey:   "h1",
		}))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	rs, err := client.ShareGame(expectedAupack, expectedRoom)
	require.NoError(t, err)
	require.NotNil(t, rs)
	assert.Equal(t, "s1", rs.SessionID)
	assert.Equal(t, "h1", rs.HostKey)
}
