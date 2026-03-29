package core

type IPCLobbyInfo struct {
	HasClient   bool   `json:"HasClient"`
	IsConnected bool   `json:"IsConnected"`
	GameState   string `json:"GameState"`
	LobbyCode   string `json:"LobbyCode"`
	ServerIP    string `json:"ServerIp"`
	ServerPort  int    `json:"ServerPort"`
	IsHost      *bool  `json:"IsHost"`
	IsInGame    *bool  `json:"IsInGame"`
}
