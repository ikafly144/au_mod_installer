package rest

import "time"

const (
	JoinGameErrorInvalidSession  = "invalid_session"
	JoinGameErrorSessionNotFound = "session_not_found"
	JoinGameErrorSessionExpired  = "session_expired"

	JoinLobbyErrorInvalidSession  = "invalid_session"
	JoinLobbyErrorSessionNotFound = "session_not_found"
	JoinLobbyErrorSessionExpired  = "session_expired"
)

type HealthStatus struct {
	Status string `json:"status"`
}

type VersionInfo struct {
	Branches []BranchInfo `json:"branches"`
}

type BranchInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type RoomInfo struct {
	LobbyCode      string `json:"lobby_code,omitempty"`
	ServerIP       string `json:"server_ip,omitempty"`
	ServerPort     uint16 `json:"server_port,omitzero"`
	MatchMakerIp   string `json:"match_maker_ip,omitempty"`
	MatchMakerPort uint16 `json:"match_maker_port,omitzero"`
	GameVersion    string `json:"game_version,omitempty"`
}

type ShareGameRequest struct {
	Aupack []byte   `json:"aupack"`
	Room   RoomInfo `json:"room"`
}

type ShareGameResponse struct {
	URL       string    `json:"url"`
	SessionID string    `json:"session_id"`
	HostKey   string    `json:"host_key"`
	ExpiresAt time.Time `json:"expires_at"`
}

type JoinGameDownloadResponse struct {
	SessionID string    `json:"session_id"`
	Aupack    []byte    `json:"aupack"`
	Room      RoomInfo  `json:"room"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ShareLobbyRequest struct {
	Aupack      []byte    `json:"aupack"`
	LobbySecret string    `json:"lobby_secret"`
	Room        *RoomInfo `json:"room,omitempty"`
}

type ShareLobbyResponse struct {
	URL       string    `json:"url"`
	SessionID string    `json:"session_id"`
	HostKey   string    `json:"host_key"`
	ExpiresAt time.Time `json:"expires_at"`
}

type UpdateLobbyRoomRequest struct {
	HostKey string    `json:"host_key"`
	Room    *RoomInfo `json:"room,omitempty"`
}

type HeartbeatLobbyRequest struct {
	SessionID string `json:"session_id"`
	HostKey   string `json:"host_key"`
}

type JoinLobbyDownloadResponse struct {
	SessionID   string    `json:"session_id"`
	Aupack      []byte    `json:"aupack"`
	LobbySecret string    `json:"lobby_secret"`
	Room        *RoomInfo `json:"room,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
}
