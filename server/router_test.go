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

func TestRouter_ShareLobby_Lifecycle(t *testing.T) {
	srv := service.NewModService(nil)
	handler := router(srv, staticVersionInfoProvider{}, "", "")

	// 1. Create shared lobby
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("aupack", "test.aupack")
	require.NoError(t, err)
	_, err = part.Write([]byte("lobby-pack-data"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("lobby_code", "LOBBY1"))
	require.NoError(t, writer.Close())

	createReq := httptest.NewRequest(http.MethodPost, "/share_lobby", body)
	createReq.Header.Set("Content-Type", writer.FormDataContentType())
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code)

	var shareRs restcommon.ShareLobbyResponse
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &shareRs))
	assert.NotEmpty(t, shareRs.SessionID)
	assert.NotEmpty(t, shareRs.HostKey)
	assert.Contains(t, shareRs.URL, "/join_lobby?session_id="+shareRs.SessionID)

	// 2. Fetch lobby download (initially has room LOBBY1)
	dlReq := httptest.NewRequest(http.MethodGet, "/join_lobby?session_id="+shareRs.SessionID+"&download=1", nil)
	dlRec := httptest.NewRecorder()
	handler.ServeHTTP(dlRec, dlReq)
	require.Equal(t, http.StatusOK, dlRec.Code)
	var dlRs restcommon.JoinLobbyDownloadResponse
	require.NoError(t, json.Unmarshal(dlRec.Body.Bytes(), &dlRs))
	require.NotNil(t, dlRs.Room)
	assert.Equal(t, "LOBBY1", dlRs.Room.LobbyCode)

	// 3. Update room dynamically without changing session ID / Lobby URL
	updateBody := &bytes.Buffer{}
	upWriter := multipart.NewWriter(updateBody)
	require.NoError(t, upWriter.WriteField("host_key", shareRs.HostKey))
	require.NoError(t, upWriter.WriteField("lobby_code", "LOBBY2_NEW"))
	require.NoError(t, upWriter.Close())

	upReq := httptest.NewRequest(http.MethodPut, "/share_lobby/"+shareRs.SessionID+"/room", updateBody)
	upReq.Header.Set("Content-Type", upWriter.FormDataContentType())
	upRec := httptest.NewRecorder()
	handler.ServeHTTP(upRec, upReq)
	require.Equal(t, http.StatusOK, upRec.Code)

	// 4. Verify updated room info
	dlReq2 := httptest.NewRequest(http.MethodGet, "/join_lobby?session_id="+shareRs.SessionID+"&download=1", nil)
	dlRec2 := httptest.NewRecorder()
	handler.ServeHTTP(dlRec2, dlReq2)
	require.Equal(t, http.StatusOK, dlRec2.Code)
	var dlRs2 restcommon.JoinLobbyDownloadResponse
	require.NoError(t, json.Unmarshal(dlRec2.Body.Bytes(), &dlRs2))
	require.NotNil(t, dlRs2.Room)
	assert.Equal(t, "LOBBY2_NEW", dlRs2.Room.LobbyCode)

	// 5. HTML join redirect page
	joinReq := httptest.NewRequest(http.MethodGet, "/join_lobby?session_id="+shareRs.SessionID, nil)
	joinRec := httptest.NewRecorder()
	handler.ServeHTTP(joinRec, joinReq)
	require.Equal(t, http.StatusOK, joinRec.Code)
	assert.Contains(t, joinRec.Body.String(), "mod-of-us://join_lobby/v1/"+shareRs.SessionID+"?server=")

	// 6. Delete shared lobby
	delReq := httptest.NewRequest(http.MethodDelete, "/share_lobby/"+shareRs.SessionID+"?host_key="+shareRs.HostKey, nil)
	delRec := httptest.NewRecorder()
	handler.ServeHTTP(delRec, delReq)
	require.Equal(t, http.StatusOK, delRec.Code)
}

func TestBuildJoinLobbyDeepLink(t *testing.T) {
	assert.Equal(
		t,
		"mod-of-us://join_lobby/v1/test-lobby-session?server=http%3A%2F%2Flocalhost%3A8080",
		buildJoinLobbyDeepLink("http://localhost:8080", "test-lobby-session", ""),
	)
	assert.Equal(
		t,
		"mod-of-us://join_lobby/v1/test-lobby-session?error_type=invalid_session&server=http%3A%2F%2Flocalhost%3A8080",
		buildJoinLobbyDeepLink("http://localhost:8080", "test-lobby-session", restcommon.JoinLobbyErrorInvalidSession),
	)
}

type mockDiscordLobbyClient struct {
	createdHostUserID uint64
	addedLobbyID      uint64
	addedUserID       uint64
	removedLobbyID    uint64
	removedUserID     uint64
	deletedLobbyID    uint64
	updatedLobbyID    uint64
	updatedMeta       map[string]string
}

func (m *mockDiscordLobbyClient) CreateLobby(ctx context.Context, hostUserID uint64, metadata map[string]string) (uint64, error) {
	m.createdHostUserID = hostUserID
	return 1234567890, nil
}

func (m *mockDiscordLobbyClient) UpdateLobbyMetadata(ctx context.Context, lobbyID uint64, metadata map[string]string) error {
	m.updatedLobbyID = lobbyID
	m.updatedMeta = metadata
	return nil
}

func (m *mockDiscordLobbyClient) AddMember(ctx context.Context, lobbyID uint64, userID uint64, metadata map[string]string) error {
	m.addedLobbyID = lobbyID
	m.addedUserID = userID
	return nil
}

func (m *mockDiscordLobbyClient) RemoveMember(ctx context.Context, lobbyID uint64, userID uint64) error {
	m.removedLobbyID = lobbyID
	m.removedUserID = userID
	return nil
}

func (m *mockDiscordLobbyClient) DeleteLobby(ctx context.Context, lobbyID uint64) error {
	m.deletedLobbyID = lobbyID
	return nil
}

func TestRouter_ShareLobby_WithDiscordLobbyIntegration(t *testing.T) {
	mockDiscord := &mockDiscordLobbyClient{}
	srv := service.NewModServiceWithOptions(nil, mockDiscord)
	handler := router(srv, staticVersionInfoProvider{}, "", "")

	// 1. Create shared lobby with host discord user id
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("aupack", "test.aupack")
	require.NoError(t, err)
	_, err = part.Write([]byte("lobby-pack-data"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("host_discord_user_id", "987654321"))
	require.NoError(t, writer.Close())

	createReq := httptest.NewRequest(http.MethodPost, "/share_lobby", body)
	createReq.Header.Set("Content-Type", writer.FormDataContentType())
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code)

	var shareRs restcommon.ShareLobbyResponse
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &shareRs))
	assert.Equal(t, uint64(1234567890), shareRs.DiscordLobbyID)
	assert.Equal(t, uint64(987654321), mockDiscord.createdHostUserID)

	// 2. Add guest member
	memberBody := []byte(`{"discord_user_id": 555555555}`)
	memberReq := httptest.NewRequest(http.MethodPost, "/share_lobby/"+shareRs.SessionID+"/members", bytes.NewReader(memberBody))
	memberReq.Header.Set("Content-Type", "application/json")
	memberRec := httptest.NewRecorder()
	handler.ServeHTTP(memberRec, memberReq)
	require.Equal(t, http.StatusOK, memberRec.Code)
	assert.Equal(t, uint64(1234567890), mockDiscord.addedLobbyID)
	assert.Equal(t, uint64(555555555), mockDiscord.addedUserID)

	// 3. Remove guest member
	delMemberReq := httptest.NewRequest(http.MethodDelete, "/share_lobby/"+shareRs.SessionID+"/members/555555555", nil)
	delMemberRec := httptest.NewRecorder()
	handler.ServeHTTP(delMemberRec, delMemberReq)
	require.Equal(t, http.StatusOK, delMemberRec.Code)
	assert.Equal(t, uint64(1234567890), mockDiscord.removedLobbyID)
	assert.Equal(t, uint64(555555555), mockDiscord.removedUserID)
}
