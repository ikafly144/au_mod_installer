package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/url"

	"github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/common/rest/model"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
)

func (c *clientImpl) GetModIDs(limit int, after string, before string) ([]string, error) {
	var mods model.ModListResult

	values := make(url.Values)
	if limit > 0 {
		values.Set("limit", fmt.Sprint(limit))
	}
	if after != "" {
		values.Set("after", after)
	}
	if before != "" {
		values.Set("before", before)
	}

	err := c.do(rest.EndpointGetModList.Compile(values, nil), nil, &mods, 1)
	return mods.IDs, err
}

func (c *clientImpl) GetMod(modID string) (*modmgr.Mod, error) {
	var mod model.ModDetails
	err := c.do(rest.EndpointGetModDetail.Compile(nil, modID), nil, &mod, 1)
	return &modmgr.Mod{ModDetails: mod}, err
}

func (c *clientImpl) GetModVersionIDs(modID string, limit int, after string) ([]string, error) {
	var versions model.ModVersionListResult
	err := c.do(rest.EndpointGetModVersionList.Compile(nil, modID), nil, &versions, 1)
	return versions.IDs, err
}

func (c *clientImpl) GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error) {
	var modVersion modmgr.ModVersion
	err := c.do(rest.EndpointGetModVersionDetail.Compile(nil, modID, versionID), nil, &modVersion, 1)
	return &modVersion, err
}

func (c *clientImpl) GetModThumbnail(modID string) ([]byte, error) {
	var thumbnail []byte
	err := c.do(rest.EndpointGetModThumbnail.Compile(nil, modID), nil, &thumbnail, 1)
	return thumbnail, err
}

func (c *clientImpl) GetLatestModVersion(modID string) (*modmgr.ModVersion, error) {
	mod, err := c.GetMod(modID)
	if err != nil {
		return nil, err
	}
	if mod.LatestVersionID == "" {
		return nil, fmt.Errorf("mod %s does not have a latest version", modID)
	}
	return c.GetModVersion(modID, mod.LatestVersionID)
}

func (c *clientImpl) CheckForUpdates(installedVersions map[string]string) (map[string]*modmgr.ModVersion, error) {
	updates := make(map[string]*modmgr.ModVersion)
	for modID, currentVersion := range installedVersions {
		latest, err := c.GetLatestModVersion(modID)
		if err != nil {
			return nil, fmt.Errorf("failed to check for updates for mod %s: %w", modID, err)
		}
		if latest != nil && latest.VersionID != currentVersion {
			updates[modID] = latest
		}
	}
	return updates, nil
}

func (c *clientImpl) ShareGame(aupack []byte, room rest.RoomInfo) (*rest.ShareGameResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	aupackPart, err := writer.CreateFormFile("aupack", "profile.aupack")
	if err != nil {
		return nil, err
	}
	if _, err := aupackPart.Write(aupack); err != nil {
		return nil, err
	}
	if err := writer.WriteField("lobby_code", room.LobbyCode); err != nil {
		return nil, err
	}
	if err := writer.WriteField("server_ip", room.ServerIP); err != nil {
		return nil, err
	}
	if err := writer.WriteField("server_port", fmt.Sprint(room.ServerPort)); err != nil {
		return nil, err
	}
	if err := writer.WriteField("match_maker_ip", room.MatchMakerIp); err != nil {
		return nil, err
	}
	if err := writer.WriteField("match_maker_port", fmt.Sprint(room.MatchMakerPort)); err != nil {
		return nil, err
	}
	if room.GameVersion != "" {
		if err := writer.WriteField("game_version", room.GameVersion); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	var rs rest.ShareGameResponse
	err = c.do(rest.EndpointShareGame.Compile(nil), encodedRequestBody{
		ContentType: writer.FormDataContentType(),
		Body:        body.Bytes(),
	}, &rs, 1)
	if err != nil {
		return nil, err
	}
	return &rs, nil
}

func (c *clientImpl) DeleteSharedGame(sessionID, hostKey string) error {
	values := make(url.Values)
	values.Set("session_id", sessionID)
	values.Set("host_key", hostKey)
	return c.do(rest.EndpointDeleteShareGame.Compile(values), nil, nil, 1)
}

func (c *clientImpl) UpdateSharedGameExpiration(sessionID, hostKey string) (*rest.ShareGameResponse, error) {
	values := make(url.Values)
	values.Set("session_id", sessionID)
	values.Set("host_key", hostKey)
	var rs rest.ShareGameResponse
	if err := c.do(rest.EndpointUpdateShareGame.Compile(values), nil, &rs, 1); err != nil {
		return nil, err
	}
	return &rs, nil
}

func (c *clientImpl) GetJoinGameDownload(sessionID string) (*rest.JoinGameDownloadResponse, error) {
	values := make(url.Values)
	values.Set("session_id", sessionID)
	values.Set("download", "1")
	var rs rest.JoinGameDownloadResponse
	if err := c.do(rest.EndpointJoinGame.Compile(values), nil, &rs, 1); err != nil {
		return nil, err
	}
	return &rs, nil
}

func (c *clientImpl) ShareLobby(aupack []byte, lobbySecret string, hostDiscordUserID uint64, room *rest.RoomInfo) (*rest.ShareLobbyResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("aupack", "profile.aupack")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(aupack); err != nil {
		return nil, err
	}
	if err := writer.WriteField("lobby_secret", lobbySecret); err != nil {
		return nil, err
	}
	if hostDiscordUserID != 0 {
		if err := writer.WriteField("host_discord_user_id", fmt.Sprint(hostDiscordUserID)); err != nil {
			return nil, err
		}
	}
	if room != nil {
		if err := writer.WriteField("lobby_code", room.LobbyCode); err != nil {
			return nil, err
		}
		if err := writer.WriteField("server_ip", room.ServerIP); err != nil {
			return nil, err
		}
		if err := writer.WriteField("server_port", fmt.Sprint(room.ServerPort)); err != nil {
			return nil, err
		}
		if err := writer.WriteField("match_maker_ip", room.MatchMakerIp); err != nil {
			return nil, err
		}
		if err := writer.WriteField("match_maker_port", fmt.Sprint(room.MatchMakerPort)); err != nil {
			return nil, err
		}
		if room.GameVersion != "" {
			if err := writer.WriteField("game_version", room.GameVersion); err != nil {
				return nil, err
			}
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	var rs rest.ShareLobbyResponse
	err = c.do(rest.EndpointShareLobby.Compile(nil), encodedRequestBody{
		ContentType: writer.FormDataContentType(),
		Body:        body.Bytes(),
	}, &rs, 1)
	if err != nil {
		return nil, err
	}
	return &rs, nil
}

func (c *clientImpl) UpdateSharedLobbyRoom(sessionID, hostKey string, room *rest.RoomInfo) (*rest.ShareLobbyResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("host_key", hostKey); err != nil {
		return nil, err
	}
	if room == nil {
		if err := writer.WriteField("clear", "true"); err != nil {
			return nil, err
		}
	} else {
		if err := writer.WriteField("lobby_code", room.LobbyCode); err != nil {
			return nil, err
		}
		if err := writer.WriteField("server_ip", room.ServerIP); err != nil {
			return nil, err
		}
		if err := writer.WriteField("server_port", fmt.Sprint(room.ServerPort)); err != nil {
			return nil, err
		}
		if err := writer.WriteField("match_maker_ip", room.MatchMakerIp); err != nil {
			return nil, err
		}
		if err := writer.WriteField("match_maker_port", fmt.Sprint(room.MatchMakerPort)); err != nil {
			return nil, err
		}
		if room.GameVersion != "" {
			if err := writer.WriteField("game_version", room.GameVersion); err != nil {
				return nil, err
			}
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	var rs rest.ShareLobbyResponse
	err := c.do(rest.EndpointUpdateLobbyRoom.Compile(nil, sessionID), encodedRequestBody{
		ContentType: writer.FormDataContentType(),
		Body:        body.Bytes(),
	}, &rs, 1)
	if err != nil {
		return nil, err
	}
	return &rs, nil
}

func (c *clientImpl) UpdateSharedLobbyExpiration(sessionID, hostKey string) (*rest.ShareLobbyResponse, error) {
	values := make(url.Values)
	values.Set("session_id", sessionID)
	values.Set("host_key", hostKey)
	var rs rest.ShareLobbyResponse
	if err := c.do(rest.EndpointHeartbeatLobby.Compile(values), nil, &rs, 1); err != nil {
		return nil, err
	}
	return &rs, nil
}

func (c *clientImpl) DeleteSharedLobby(sessionID, hostKey string) error {
	values := make(url.Values)
	values.Set("host_key", hostKey)
	return c.do(rest.EndpointDeleteLobby.Compile(values, sessionID), nil, nil, 1)
}

func (c *clientImpl) GetJoinLobbyDownload(sessionID string) (*rest.JoinLobbyDownloadResponse, error) {
	values := make(url.Values)
	values.Set("session_id", sessionID)
	values.Set("download", "1")
	var rs rest.JoinLobbyDownloadResponse
	if err := c.do(rest.EndpointJoinLobby.Compile(values), nil, &rs, 1); err != nil {
		return nil, err
	}
	return &rs, nil
}

func (c *clientImpl) AddLobbyMember(sessionID string, discordUserID uint64) error {
	body := rest.JoinLobbyMemberRequest{
		DiscordUserID: discordUserID,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(rest.EndpointAddLobbyMember.Compile(nil, sessionID), encodedRequestBody{
		ContentType: "application/json",
		Body:        bodyBytes,
	}, nil, 1)
}

func (c *clientImpl) RemoveLobbyMember(sessionID string, discordUserID uint64) error {
	return c.do(rest.EndpointRemoveLobbyMember.Compile(nil, sessionID, fmt.Sprint(discordUserID)), nil, nil, 1)
}
