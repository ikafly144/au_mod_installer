package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ikafly144/au_mod_installer/server/service"
)

func TestRouter_ShareGame_AcceptsMultipartFormData(t *testing.T) {
	srv := service.NewModService(nil)
	handler := router(srv, "", "")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("aupack", "test.aupack")
	require.NoError(t, err)
	_, err = part.Write([]byte("pack-data"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("lobby_code", "ABCD"))
	require.NoError(t, writer.WriteField("server_ip", "127.0.0.1"))
	require.NoError(t, writer.WriteField("server_port", "22023"))
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
}

func TestRouter_ShareGame_RejectsInvalidServerPort(t *testing.T) {
	srv := service.NewModService(nil)
	handler := router(srv, "", "")

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
