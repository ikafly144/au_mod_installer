package rest

import (
	"errors"

	"github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

type OfflineClient struct{}

var _ Client = (*OfflineClient)(nil)

func NewOfflineClient() *OfflineClient {
	return &OfflineClient{}
}

func (c *OfflineClient) GetHealthStatus() (*rest.HealthStatus, error) {
	return &rest.HealthStatus{
		Status: "offline",
	}, nil
}

func (c *OfflineClient) GetVersionInfo() (*rest.VersionInfo, error) {
	return nil, errors.New("offline mode: version info not available")
}

func (c *OfflineClient) ServerBaseURL() string {
	return ""
}

func (c *OfflineClient) GetModIDs(limit int, after string, before string) ([]string, error) {
	return nil, errors.New("offline mode: mod IDs not available")
}

func (c *OfflineClient) GetMod(modID string) (*modmgr.Mod, error) {
	return nil, errors.New("offline mode: mod details not available")
}

func (c *OfflineClient) GetModVersionIDs(modID string, limit int, after string) ([]string, error) {
	return nil, errors.New("offline mode: mod versions not available")
}

func (c *OfflineClient) GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error) {
	return nil, errors.New("offline mode: mod version details not available")
}

func (c *OfflineClient) GetLatestModVersion(modID string) (*modmgr.ModVersion, error) {
	return nil, errors.New("offline mode: latest mod version details not available")
}

func (c *OfflineClient) CheckForUpdates(installedVersions map[string]string) (map[string]*modmgr.ModVersion, error) {
	return nil, errors.New("offline mode: update check not available")
}

func (c *OfflineClient) GetModThumbnail(modID string) ([]byte, error) {
	return nil, errors.New("offline mode: thumbnail not available")
}

func (c *OfflineClient) ShareGame(aupack []byte, room rest.RoomInfo) (*rest.ShareGameResponse, error) {
	return nil, errors.New("offline mode: share game not available")
}

func (c *OfflineClient) DeleteSharedGame(sessionID, hostKey string) error {
	return errors.New("offline mode: delete shared game not available")
}

func (c *OfflineClient) UpdateSharedGameExpiration(sessionID, hostKey string) (*rest.ShareGameResponse, error) {
	return nil, errors.New("offline mode: update shared game expiration not available")
}

func (c *OfflineClient) GetJoinGameDownload(sessionID string) (*rest.JoinGameDownloadResponse, error) {
	return nil, errors.New("offline mode: join game download not available")
}

func (c *OfflineClient) ShareLobby(aupack []byte, lobbySecret string, hostDiscordUserID uint64, room *rest.RoomInfo) (*rest.ShareLobbyResponse, error) {
	return nil, errors.New("offline mode: share lobby not available")
}

func (c *OfflineClient) UpdateSharedLobbyRoom(sessionID, hostKey string, room *rest.RoomInfo) (*rest.ShareLobbyResponse, error) {
	return nil, errors.New("offline mode: update shared lobby room not available")
}

func (c *OfflineClient) UpdateSharedLobbyExpiration(sessionID, hostKey string) (*rest.ShareLobbyResponse, error) {
	return nil, errors.New("offline mode: update shared lobby expiration not available")
}

func (c *OfflineClient) DeleteSharedLobby(sessionID, hostKey string) error {
	return errors.New("offline mode: delete shared lobby not available")
}

func (c *OfflineClient) GetJoinLobbyDownload(sessionID string) (*rest.JoinLobbyDownloadResponse, error) {
	return nil, errors.New("offline mode: join lobby download not available")
}

func (c *OfflineClient) AddLobbyMember(sessionID string, discordUserID uint64) error {
	return errors.New("offline mode: add lobby member not available")
}
