package rest

import "time"

type HealthStatus struct {
	Status           string   `json:"status"`
	WorkingVersion   string   `json:"working_version,omitempty"`
	DisabledVersions []string `json:"disabled_versions,omitempty"`
}

type RoomInfo struct {
	LobbyCode  string `json:"lobby_code,omitempty"`
	ServerIP   string `json:"server_ip,omitempty"`
	ServerPort uint16 `json:"server_port,omitempty"`
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
