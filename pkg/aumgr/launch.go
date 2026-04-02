package aumgr

const (
	EpicCatalogId  = "729a86a5146640a2ace9e8c595414c56"
	EpicNamespace  = "33956bcb55d4452d8c47e16b94e294bd"
	EpicArtifactId = "963137e4c29d4c79a81323b8fab03a40"
)

type DirectJoinInfo struct {
	LobbyCode      string
	ServerIP       string
	ServerPort     uint16
	MatchMakerIp   string
	MatchMakerPort uint16
}
