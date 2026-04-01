package core

import (
	"fmt"
	"strings"
)

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

func (i *IPCLobbyInfo) String() string {
	b := &strings.Builder{}
	fmt.Fprint(b, "IPCLobbyInfo{")
	fmt.Fprintf(b, "HasClient: %t, ", i.HasClient)
	fmt.Fprintf(b, "IsConnected: %t, ", i.IsConnected)
	fmt.Fprintf(b, "GameState: %s, ", i.GameState)
	fmt.Fprintf(b, "LobbyCode: %s, ", i.LobbyCode)
	fmt.Fprintf(b, "ServerIP: %s, ", i.ServerIP)
	fmt.Fprintf(b, "ServerPort: %d, ", i.ServerPort)
	if i.IsHost != nil {
		fmt.Fprintf(b, "IsHost: %t", *i.IsHost)
	} else {
		fmt.Fprintln(b, "IsHost: nil")
	}
	fmt.Fprint(b, ", ")
	if i.IsInGame != nil {
		fmt.Fprintf(b, "IsInGame: %t", *i.IsInGame)
	} else {
		fmt.Fprintln(b, "IsInGame: nil")
	}
	fmt.Fprint(b, "}")
	return b.String()
}
