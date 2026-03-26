package profile

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeSharedArchive(t *testing.T) {
	updatedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	source := SharedProfile{
		ID:          uuid.New(),
		Name:        "Shared",
		Author:      "tester",
		Description: "desc",
		ModVersions: map[string]string{
			"mod-a": "1.0.0",
		},
		UpdatedAt: updatedAt,
	}
	icon := []byte{0x89, 0x50, 0x4e, 0x47}

	archive, err := buildProfileArchive(t, source, icon, "/mod-of-us.profile.json", "/icon.png")
	require.NoError(t, err)

	decoded, decodedIcon, err := decodeSharedArchiveBytes(archive)
	require.NoError(t, err)
	require.NotNil(t, decoded)
	assert.Equal(t, source.ID, decoded.ID)
	assert.Equal(t, source.Name, decoded.Name)
	assert.Equal(t, source.Author, decoded.Author)
	assert.Equal(t, source.Description, decoded.Description)
	assert.Equal(t, source.ModVersions, decoded.ModVersions)
	assert.True(t, source.UpdatedAt.Equal(decoded.UpdatedAt))
	assert.Equal(t, icon, decodedIcon)
}

func TestDecodeSharedArchive_WithoutIcon(t *testing.T) {
	source := SharedProfile{
		ID:   uuid.New(),
		Name: "No icon",
	}
	archive, err := buildProfileArchive(t, source, nil, "mod-of-us.profile.json", "")
	require.NoError(t, err)

	decoded, decodedIcon, err := decodeSharedArchiveBytes(archive)
	require.NoError(t, err)
	require.NotNil(t, decoded)
	assert.Equal(t, source.ID, decoded.ID)
	assert.Nil(t, decodedIcon)
}

func TestDecodeSharedArchive_MissingSharedProfile(t *testing.T) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	w, err := zw.Create(SharedArchiveProfilePath)
	require.NoError(t, err)
	_, err = w.Write([]byte(`{"not_sharedprofile":{}}`))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	decoded, icon, err := decodeSharedArchiveBytes(buf.Bytes())
	require.Error(t, err)
	assert.Nil(t, decoded)
	assert.Nil(t, icon)
}

func TestDecodeSharedArchive_EmptyReader(t *testing.T) {
	decoded, icon, err := decodeSharedArchiveBytes(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open profile archive")
	assert.Nil(t, decoded)
	assert.Nil(t, icon)
}

func TestDecodeSharedArchive_NilReader(t *testing.T) {
	decoded, icon, err := DecodeSharedArchive(nil, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "archive reader is nil")
	assert.Nil(t, decoded)
	assert.Nil(t, icon)
}

func TestDecodeSharedArchive_ReaderError(t *testing.T) {
	decoded, icon, err := DecodeSharedArchive(errorReaderAt{}, 1024)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "readat failed")
	assert.Nil(t, decoded)
	assert.Nil(t, icon)
}

type errorReaderAt struct{}

func (errorReaderAt) ReadAt(_ []byte, _ int64) (int, error) {
	return 0, errors.New("readat failed")
}

func decodeSharedArchiveBytes(data []byte) (*SharedProfile, []byte, error) {
	return DecodeSharedArchive(bytes.NewReader(data), int64(len(data)))
}

func buildProfileArchive(t *testing.T, shared SharedProfile, icon []byte, profilePath, iconPath string) ([]byte, error) {
	t.Helper()

	document := map[string]any{
		"sharedprofile": shared,
	}
	jsonBody, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	profileWriter, err := zw.Create(profilePath)
	if err != nil {
		return nil, err
	}
	if _, err := profileWriter.Write(jsonBody); err != nil {
		return nil, err
	}

	if iconPath != "" && len(icon) > 0 {
		iconWriter, err := zw.Create(iconPath)
		if err != nil {
			return nil, err
		}
		if _, err := iconWriter.Write(icon); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
