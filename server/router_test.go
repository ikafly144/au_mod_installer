package main

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	restcommon "github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/server/service"
)

type staticVersionInfoProvider struct{}

func (staticVersionInfoProvider) GetVersionInfo(ctx context.Context) (*restcommon.VersionInfo, error) {
	return &restcommon.VersionInfo{}, nil
}

func TestRouter_ShareGame_AcceptsMultipartFormData(t *testing.T) {
	srv := service.NewModService(nil)
	handler := router(srv, staticVersionInfoProvider{}, "", "")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("aupack", "test.aupack")
	require.NoError(t, err)
	_, err = part.Write([]byte("pack-data"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("lobby_code", "ABCD"))
	require.NoError(t, writer.WriteField("server_ip", "127.0.0.1"))
	require.NoError(t, writer.WriteField("server_port", "22023"))
	require.NoError(t, writer.WriteField("game_version", "2024.3.5"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/share_game", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var rs struct {
		SessionID string `json:"session_id"`
		HostKey   string `json:"host_key"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rs))
	assert.NotEmpty(t, rs.SessionID)
	assert.NotEmpty(t, rs.HostKey)

	// Verify /join_game returns game_version
	joinReq := httptest.NewRequest(http.MethodGet, "/join_game?session_id="+rs.SessionID+"&download=1", nil)
	joinRec := httptest.NewRecorder()
	handler.ServeHTTP(joinRec, joinReq)
	require.Equal(t, http.StatusOK, joinRec.Code)
	var downloadRs restcommon.JoinGameDownloadResponse
	require.NoError(t, json.Unmarshal(joinRec.Body.Bytes(), &downloadRs))
	assert.Equal(t, "2024.3.5", downloadRs.Room.GameVersion)
}

func TestRouter_ShareGame_RejectsInvalidServerPort(t *testing.T) {
	srv := service.NewModService(nil)
	handler := router(srv, staticVersionInfoProvider{}, "", "")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("aupack", "test.aupack")
	require.NoError(t, err)
	_, err = part.Write([]byte("pack-data"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("server_port", "not-a-number"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/share_game", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestJoinGameHTML(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		html := joinGameHTML("", "mod-of-us://join_game/v1/test-session?server=http%3A%2F%2Flocalhost%3A8080", true)
		assert.Contains(t, html, "参加リンクを開いています...")
		assert.NotContains(t, html, `class="error"`)
		assert.Contains(t, html, `href="mod-of-us://join_game/v1/test-session?server=http%3A%2F%2Flocalhost%3A8080"`)
		assert.Contains(t, html, `const link="mod-of-us:\/\/join_game\/v1\/test-session?server=http%3A%2F%2Flocalhost%3A8080";`)
		assert.Contains(t, html, launcherReleaseURL)
	})

	t.Run("failure with error message", func(t *testing.T) {
		html := joinGameHTML("エラーが発生しました & <script>", "mod-of-us://join_game/v1/test-session?error_type=invalid_session&server=http%3A%2F%2Flocalhost%3A8080", false)
		assert.Contains(t, html, "参加リンクを開けませんでした。")
		assert.Contains(t, html, `<p class="error">エラーが発生しました &amp; &lt;script&gt;</p>`)
		assert.Contains(t, html, `href="mod-of-us://join_game/v1/test-session?error_type=invalid_session&amp;server=http%3A%2F%2Flocalhost%3A8080"`)
		assert.Contains(t, html, launcherReleaseURL)
	})
}

func TestBuildJoinGameDeepLink(t *testing.T) {
	assert.Equal(
		t,
		"mod-of-us://join_game/v1/test-session?server=http%3A%2F%2Flocalhost%3A8080",
		buildJoinGameDeepLink("http://localhost:8080", "test-session", ""),
	)
	assert.Equal(
		t,
		"mod-of-us://join_game/v1/test-session?error_type=invalid_session&server=http%3A%2F%2Flocalhost%3A8080",
		buildJoinGameDeepLink("http://localhost:8080", "test-session", restcommon.JoinGameErrorInvalidSession),
	)
	assert.Equal(
		t,
		"mod-of-us://join_game/v1/test-session?error_type=session_not_found&server=http%3A%2F%2Flocalhost%3A8080",
		buildJoinGameDeepLink("http://localhost:8080", "test-session", restcommon.JoinGameErrorSessionNotFound),
	)
	assert.Equal(
		t,
		"mod-of-us://join_game/v1/test-session?error_type=session_expired&server=http%3A%2F%2Flocalhost%3A8080",
		buildJoinGameDeepLink("http://localhost:8080", "test-session", restcommon.JoinGameErrorSessionExpired),
	)
}

func TestRouter_JoinGame_HTML(t *testing.T) {
	srv := service.NewModService(nil)
	handler := router(srv, staticVersionInfoProvider{}, "", "")

	t.Run("valid session returns HTML", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("aupack", "test.aupack")
		require.NoError(t, err)
		_, err = part.Write([]byte("pack-data"))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		createReq := httptest.NewRequest(http.MethodPost, "/share_game", body)
		createReq.Header.Set("Content-Type", writer.FormDataContentType())
		createRec := httptest.NewRecorder()
		handler.ServeHTTP(createRec, createReq)
		require.Equal(t, http.StatusOK, createRec.Code)

		var rs struct {
			SessionID string `json:"session_id"`
		}
		require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &rs))

		joinReq := httptest.NewRequest(http.MethodGet, "/join_game?session_id="+rs.SessionID, nil)
		joinRec := httptest.NewRecorder()
		handler.ServeHTTP(joinRec, joinReq)

		assert.Equal(t, http.StatusOK, joinRec.Code)
		assert.Equal(t, "text/html; charset=utf-8", joinRec.Header().Get("Content-Type"))
		assert.Contains(t, joinRec.Body.String(), "参加リンクを開いています...")
		assert.Contains(t, joinRec.Body.String(), "mod-of-us://join_game/v1/"+rs.SessionID+"?server=")
	})

	t.Run("non-existent session returns 404 HTML", func(t *testing.T) {
		joinReq := httptest.NewRequest(http.MethodGet, "/join_game?session_id=nonexistent", nil)
		joinRec := httptest.NewRecorder()
		handler.ServeHTTP(joinRec, joinReq)

		assert.Equal(t, http.StatusNotFound, joinRec.Code)
		assert.Equal(t, "text/html; charset=utf-8", joinRec.Header().Get("Content-Type"))
		assert.Contains(t, joinRec.Body.String(), "この参加リンクは見つかりません。")
		assert.Contains(t, joinRec.Body.String(), "参加リンクを開けませんでした。")
		assert.Contains(t, joinRec.Body.String(), "error_type=session_not_found")
	})
}
