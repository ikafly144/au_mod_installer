package uicommon

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	restcommon "github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

func TestFindBranchVersion(t *testing.T) {
	info := &restcommon.VersionInfo{
		Branches: []restcommon.BranchInfo{
			{Name: "stable", Version: "v1.2.0"},
			{Name: "preview", Version: "v1.3.0-rc.1"},
			{Name: "dev", Version: "v1.4.0-alpha.1"},
		},
	}

	assert.Equal(t, "v1.2.0", FindBranchVersion(info, "stable"))
	assert.Equal(t, "v1.2.0", FindBranchVersion(info, "STABLE"))
	assert.Equal(t, "v1.3.0-rc.1", FindBranchVersion(info, "preview"))
	assert.Equal(t, "v1.4.0-alpha.1", FindBranchVersion(info, "dev"))
	assert.Equal(t, "", FindBranchVersion(info, "canary"))
	assert.Equal(t, "", FindBranchVersion(nil, "stable"))
}

type mockRestClient struct {
	versionInfo *restcommon.VersionInfo
	err         error
}

func (m *mockRestClient) ServerBaseURL() string {
	return "http://localhost"
}

func (m *mockRestClient) GetHealthStatus() (*restcommon.HealthStatus, error) {
	return &restcommon.HealthStatus{Status: "ok"}, nil
}

func (m *mockRestClient) GetVersionInfo() (*restcommon.VersionInfo, error) {
	return m.versionInfo, m.err
}

func (m *mockRestClient) GetModIDs(limit int, after string, before string) ([]string, error) {
	return nil, nil
}

func (m *mockRestClient) GetMod(modID string) (*modmgr.Mod, error) {
	return nil, nil
}

func (m *mockRestClient) GetModVersionIDs(modID string, limit int, after string) ([]string, error) {
	return nil, nil
}

func (m *mockRestClient) GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error) {
	return nil, nil
}

func (m *mockRestClient) GetLatestModVersion(modID string) (*modmgr.ModVersion, error) {
	return nil, nil
}

func (m *mockRestClient) GetModThumbnail(modID string) ([]byte, error) {
	return nil, nil
}

func (m *mockRestClient) CheckForUpdates(installedVersions map[string]string) (map[string]*modmgr.ModVersion, error) {
	return nil, nil
}

func (m *mockRestClient) ShareGame(aupack []byte, room restcommon.RoomInfo) (*restcommon.ShareGameResponse, error) {
	return nil, nil
}

func (m *mockRestClient) UpdateSharedGameExpiration(sessionID, hostKey string) (*restcommon.ShareGameResponse, error) {
	return nil, nil
}

func (m *mockRestClient) DeleteSharedGame(sessionID, hostKey string) error {
	return nil
}

func (m *mockRestClient) GetJoinGameDownload(sessionID string) (*restcommon.JoinGameDownloadResponse, error) {
	return nil, nil
}

func TestCheckForUpdatesNoUpdate(t *testing.T) {
	mock := &mockRestClient{
		versionInfo: &restcommon.VersionInfo{
			Branches: []restcommon.BranchInfo{
				{Name: "stable", Version: "v1.0.0"},
			},
		},
	}

	state := &State{
		Version: "v1.0.0",
		Rest:    mock,
	}

	state.CheckForUpdates(context.Background(), false)
}

func TestFormatBytes(t *testing.T) {
	assert.Equal(t, "500 B", formatBytes(500))
	assert.Equal(t, "1.0 KB", formatBytes(1024))
	assert.Equal(t, "1.5 MB", formatBytes(1572864))
	assert.Equal(t, "10.0 MB", formatBytes(10485760))
}

func TestResolveLatestUpdateTag(t *testing.T) {
	mock := &mockRestClient{
		versionInfo: &restcommon.VersionInfo{
			Branches: []restcommon.BranchInfo{
				{Name: "stable", Version: "v1.2.0"},
			},
		},
	}

	state := &State{
		Version: "v1.0.0",
		Rest:    mock,
	}

	// When newer version exists on server, returns the newer version
	assert.Equal(t, "v1.2.0", state.ResolveLatestUpdateTag("v1.1.0"))

	// When tag passed is already newer than or equal to server, preserves tag
	assert.Equal(t, "v1.3.0", state.ResolveLatestUpdateTag("v1.3.0"))
	assert.Equal(t, "v1.2.0", state.ResolveLatestUpdateTag("v1.2.0"))

	// When initial tag is empty, resolves to server version
	assert.Equal(t, "v1.2.0", state.ResolveLatestUpdateTag(""))

	// When Rest is nil, returns original tag
	stateNoRest := &State{
		Version: "v1.0.0",
		Rest:    nil,
	}
	assert.Equal(t, "v1.1.0", stateNoRest.ResolveLatestUpdateTag("v1.1.0"))
}

func TestCheckAvailableUpdate(t *testing.T) {
	mock := &mockRestClient{
		versionInfo: &restcommon.VersionInfo{
			Branches: []restcommon.BranchInfo{
				{Name: "stable", Version: "v1.2.0"},
			},
		},
	}

	state := &State{
		Version: "v1.0.0",
		Rest:    mock,
	}

	tag, isMandatory, err := state.CheckAvailableUpdate()
	assert.NoError(t, err)
	assert.Equal(t, "v1.2.0", tag)
	assert.True(t, isMandatory)

	// Up to date
	stateUpToDate := &State{
		Version: "v1.2.0",
		Rest:    mock,
	}
	tagUpToDate, isMandatoryUpToDate, errUpToDate := stateUpToDate.CheckAvailableUpdate()
	assert.NoError(t, errUpToDate)
	assert.Equal(t, "", tagUpToDate)
	assert.False(t, isMandatoryUpToDate)

	// Offline / no rest
	stateOffline := &State{
		Version: "v1.0.0",
		Rest:    nil,
	}
	_, _, errOffline := stateOffline.CheckAvailableUpdate()
	assert.Error(t, errOffline)
}


