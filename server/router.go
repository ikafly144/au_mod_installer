package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ikafly144/au_mod_installer/common/rest"
	restmodel "github.com/ikafly144/au_mod_installer/common/rest/model"
	"github.com/ikafly144/au_mod_installer/server/service"
)

func router(srv *service.ModService, versionProvider service.VersionInfoProvider, pathPrefix string, basePath string) http.Handler {
	r := gin.Default()

	api := r.Group(basePath)
	api.GET(rest.EndpointGetModList.Route, func(ctx *gin.Context) {
		after := ctx.Query("after")
		limitStr := ctx.Query("limit")
		limit := 0
		if limitStr != "" {
			var err error
			limit, err = strconv.Atoi(limitStr)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
				return
			}
		}

		modIDs, nextID, err := srv.GetModIds(after, limit)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get mod IDs", "error", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get mod IDs"})
			return
		}

		ctx.JSON(http.StatusOK, restmodel.ModListResult{
			IDs:    modIDs,
			NextID: nextID,
		})
	})
	api.GET(rest.EndpointGetModDetail.Route, func(ctx *gin.Context) {
		modID := ctx.Param("mod_id")

		details, err := srv.GetModDetails(modID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get mod details", "mod_id", modID, "error", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get mod details"})
			return
		}

		ctx.JSON(http.StatusOK, details)
	})
	api.GET(rest.EndpointGetModVersionList.Route, func(ctx *gin.Context) {
		modID := ctx.Param("mod_id")

		versionIDs, err := srv.GetModVersionIds(modID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get mod version IDs", "mod_id", modID, "error", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get mod version IDs"})
			return
		}

		ctx.JSON(http.StatusOK, restmodel.ModVersionListResult{
			IDs: versionIDs,
		})
	})
	api.GET(rest.EndpointGetModVersionDetail.Route, func(ctx *gin.Context) {
		modID := ctx.Param("mod_id")
		versionID := ctx.Param("version_id")

		details, err := srv.GetModVersionDetails(modID, versionID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get mod version details", "mod_id", modID, "version_id", versionID, "error", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get mod version details"})
			return
		}

		ctx.JSON(http.StatusOK, details)
	})
	api.GET(rest.EndpointGetModThumbnail.Route, func(ctx *gin.Context) {
		modID := ctx.Param("mod_id")

		modDetails, err := srv.GetModDetails(modID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get mod details", "mod_id", modID, "error", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get mod details"})
			return
		}

		thumbnailURI := modDetails.ThumbnailURI

		if thumbnailURI == nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Thumbnail not found"})
			return
		}

		ctx.Redirect(http.StatusFound, *thumbnailURI)
	})
	api.GET(rest.EndpointHealth.Route, func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	api.GET(rest.EndpointGetVersionInfo.Route, func(ctx *gin.Context) {
		if versionProvider == nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "version info not available"})
			return
		}
		info, err := versionProvider.GetVersionInfo(ctx.Request.Context())
		if err != nil {
			slog.ErrorContext(ctx.Request.Context(), "Failed to get version info", "error", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get version info"})
			return
		}
		ctx.JSON(http.StatusOK, info)
	})
	api.POST(rest.EndpointShareGame.Route, func(ctx *gin.Context) {
		fileHeader, err := ctx.FormFile("aupack")
		if err != nil {
			if errors.Is(err, http.ErrMissingFile) {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "aupack is required"})
				return
			}
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart body"})
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid aupack file"})
			return
		}
		defer file.Close()

		aupack, err := io.ReadAll(file)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "failed to read aupack file"})
			return
		}
		serverPort := uint16(0)
		serverPortStr := strings.TrimSpace(ctx.PostForm("server_port"))
		if serverPortStr != "" {
			parsed, err := strconv.ParseUint(serverPortStr, 10, 16)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid server_port"})
				return
			}
			serverPort = uint16(parsed)
		}
		matchMakerPort := uint16(0)
		matchMakerPortStr := strings.TrimSpace(ctx.PostForm("match_maker_port"))
		if matchMakerPortStr != "" {
			parsed, err := strconv.ParseUint(matchMakerPortStr, 10, 16)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid match_maker_port"})
				return
			}
			matchMakerPort = uint16(parsed)
		}
		req := rest.ShareGameRequest{
			Aupack: aupack,
			Room: rest.RoomInfo{
				LobbyCode:      strings.TrimSpace(ctx.PostForm("lobby_code")),
				ServerIP:       strings.TrimSpace(ctx.PostForm("server_ip")),
				ServerPort:     serverPort,
				MatchMakerIp:   strings.TrimSpace(ctx.PostForm("match_maker_ip")),
				MatchMakerPort: matchMakerPort,
				GameVersion:    strings.TrimSpace(ctx.PostForm("game_version")),
			},
		}
		if len(req.Aupack) == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "aupack is required"})
			return
		}
		ip := clientIP(ctx)
		rs, err := srv.CreateSharedGame(ip, req)
		if err != nil {
			if err == service.ErrShareGameRateLimited {
				ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limited"})
				return
			}
			slog.ErrorContext(ctx, "Failed to create shared game", "error", err, "ip", ip)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create shared game"})
			return
		}
		joinPath := combinePath(pathPrefix, basePath, rest.EndpointJoinGame.Route)
		rs.URL = absoluteURL(ctx, joinPath+"?session_id="+url.QueryEscape(rs.SessionID))
		ctx.JSON(http.StatusOK, rs)
	})
	api.DELETE(rest.EndpointDeleteShareGame.Route, func(ctx *gin.Context) {
		sessionID := strings.TrimSpace(ctx.Query("session_id"))
		hostKey := strings.TrimSpace(ctx.Query("host_key"))
		if sessionID == "" || hostKey == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id and host_key are required"})
			return
		}
		if err := srv.DeleteSharedGame(sessionID, hostKey); err != nil {
			switch err {
			case service.ErrShareGameNotFound:
				ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			case service.ErrShareGameUnauthorized:
				ctx.JSON(http.StatusForbidden, gin.H{"error": "invalid host key"})
			default:
				slog.ErrorContext(ctx, "Failed to delete shared game", "error", err, "session_id", sessionID)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete shared game"})
			}
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api.PUT(rest.EndpointUpdateShareGame.Route, func(ctx *gin.Context) {
		sessionID := strings.TrimSpace(ctx.Query("session_id"))
		hostKey := strings.TrimSpace(ctx.Query("host_key"))
		if sessionID == "" || hostKey == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id and host_key are required"})
			return
		}
		rs, err := srv.UpdateSharedGameExpiration(sessionID, hostKey)
		if err != nil {
			switch err {
			case service.ErrShareGameNotFound:
				ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			case service.ErrShareGameUnauthorized:
				ctx.JSON(http.StatusForbidden, gin.H{"error": "invalid host key"})
			default:
				slog.ErrorContext(ctx, "Failed to update shared game expiration", "error", err, "session_id", sessionID)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update shared game expiration"})
			}
			return
		}
		ctx.JSON(http.StatusOK, rs)
	})
	api.GET(rest.EndpointJoinGame.Route, func(ctx *gin.Context) {
		sessionID := strings.TrimSpace(ctx.Query("session_id"))
		if sessionID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}
		if ctx.Query("download") != "" {
			data, err := srv.GetJoinGameDownload(sessionID)
			if err != nil {
				switch err {
				case service.ErrShareGameNotFound:
					ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
				case service.ErrShareGameExpired:
					ctx.JSON(http.StatusGone, gin.H{"error": "session expired"})
				default:
					slog.ErrorContext(ctx, "Failed to get join game download", "error", err, "session_id", sessionID)
					ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get join game data"})
				}
				return
			}
			ctx.JSON(http.StatusOK, data)
			return
		}

		serverBase := absoluteURL(ctx, combinePath(pathPrefix, basePath, ""))
		_, err := srv.GetJoinGameMeta(sessionID)
		if err != nil {
			message := "この参加リンクは無効です。時間切れの可能性があります。"
			errorType := rest.JoinGameErrorInvalidSession
			switch err {
			case service.ErrShareGameExpired:
				message = "この参加リンクは有効期限切れです。"
				errorType = rest.JoinGameErrorSessionExpired
			case service.ErrShareGameNotFound:
				message = "この参加リンクは見つかりません。"
				errorType = rest.JoinGameErrorSessionNotFound
			}
			deepLink := buildJoinGameDeepLink(serverBase, sessionID, errorType)
			ctx.Header("Content-Type", "text/html; charset=utf-8")
			ctx.String(http.StatusNotFound, joinGameHTML(message, deepLink, false))
			return
		}
		deepLink := buildJoinGameDeepLink(serverBase, sessionID, "")
		ctx.Header("Content-Type", "text/html; charset=utf-8")
		ctx.String(http.StatusOK, joinGameHTML("", deepLink, true))
	})

	// ---------------- Lobby Sharing Endpoints ----------------

	api.POST(rest.EndpointShareLobby.Route, func(ctx *gin.Context) {
		file, _, err := ctx.Request.FormFile("aupack")
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "aupack file is required"})
			return
		}
		defer file.Close()

		aupack, err := io.ReadAll(file)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "failed to read aupack file"})
			return
		}
		lobbySecret := strings.TrimSpace(ctx.PostForm("lobby_secret"))
		if lobbySecret == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "lobby_secret is required"})
			return
		}

		var room *rest.RoomInfo
		lobbyCode := strings.TrimSpace(ctx.PostForm("lobby_code"))
		serverIP := strings.TrimSpace(ctx.PostForm("server_ip"))
		if lobbyCode != "" || serverIP != "" {
			serverPort := uint16(0)
			if serverPortStr := strings.TrimSpace(ctx.PostForm("server_port")); serverPortStr != "" {
				if parsed, err := strconv.ParseUint(serverPortStr, 10, 16); err == nil {
					serverPort = uint16(parsed)
				}
			}
			matchMakerPort := uint16(0)
			if mmPortStr := strings.TrimSpace(ctx.PostForm("match_maker_port")); mmPortStr != "" {
				if parsed, err := strconv.ParseUint(mmPortStr, 10, 16); err == nil {
					matchMakerPort = uint16(parsed)
				}
			}
			room = &rest.RoomInfo{
				LobbyCode:      lobbyCode,
				ServerIP:       serverIP,
				ServerPort:     serverPort,
				MatchMakerIp:   strings.TrimSpace(ctx.PostForm("match_maker_ip")),
				MatchMakerPort: matchMakerPort,
				GameVersion:    strings.TrimSpace(ctx.PostForm("game_version")),
			}
		}

		hostDiscordUserID := uint64(0)
		if hostDiscordUserIDStr := strings.TrimSpace(ctx.PostForm("host_discord_user_id")); hostDiscordUserIDStr != "" {
			if parsed, err := strconv.ParseUint(hostDiscordUserIDStr, 10, 64); err == nil {
				hostDiscordUserID = parsed
			}
		}

		req := rest.ShareLobbyRequest{
			Aupack:            aupack,
			LobbySecret:       lobbySecret,
			HostDiscordUserID: hostDiscordUserID,
			Room:              room,
		}

		ip := clientIP(ctx)
		rs, err := srv.CreateSharedLobby(ip, req)
		if err != nil {
			if err == service.ErrShareLobbyRateLimited {
				ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limited"})
				return
			}
			slog.ErrorContext(ctx, "Failed to create shared lobby", "error", err, "ip", ip)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create shared lobby"})
			return
		}
		joinPath := combinePath(pathPrefix, basePath, rest.EndpointJoinLobby.Route)
		rs.URL = absoluteURL(ctx, joinPath+"?session_id="+url.QueryEscape(rs.SessionID))
		ctx.JSON(http.StatusOK, rs)
	})

	api.PUT(rest.EndpointUpdateLobbyRoom.Route, func(ctx *gin.Context) {
		sessionID := strings.TrimSpace(ctx.Param("session_id"))
		if sessionID == "" {
			sessionID = strings.TrimSpace(ctx.Query("session_id"))
		}
		hostKey := strings.TrimSpace(ctx.Query("host_key"))
		if hostKey == "" {
			hostKey = strings.TrimSpace(ctx.PostForm("host_key"))
		}
		if sessionID == "" || hostKey == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id and host_key are required"})
			return
		}

		clearRoom := strings.EqualFold(strings.TrimSpace(ctx.Query("clear")), "true") || strings.EqualFold(strings.TrimSpace(ctx.PostForm("clear")), "true")
		var room *rest.RoomInfo
		if !clearRoom {
			lobbyCode := strings.TrimSpace(ctx.PostForm("lobby_code"))
			if lobbyCode == "" {
				lobbyCode = strings.TrimSpace(ctx.Query("lobby_code"))
			}
			serverIP := strings.TrimSpace(ctx.PostForm("server_ip"))
			if serverIP == "" {
				serverIP = strings.TrimSpace(ctx.Query("server_ip"))
			}
			if lobbyCode != "" || serverIP != "" {
				serverPort := uint16(0)
				serverPortStr := strings.TrimSpace(ctx.PostForm("server_port"))
				if serverPortStr == "" {
					serverPortStr = strings.TrimSpace(ctx.Query("server_port"))
				}
				if serverPortStr != "" {
					if parsed, err := strconv.ParseUint(serverPortStr, 10, 16); err == nil {
						serverPort = uint16(parsed)
					}
				}
				matchMakerPort := uint16(0)
				mmPortStr := strings.TrimSpace(ctx.PostForm("match_maker_port"))
				if mmPortStr == "" {
					mmPortStr = strings.TrimSpace(ctx.Query("match_maker_port"))
				}
				if mmPortStr != "" {
					if parsed, err := strconv.ParseUint(mmPortStr, 10, 16); err == nil {
						matchMakerPort = uint16(parsed)
					}
				}
				gameVer := strings.TrimSpace(ctx.PostForm("game_version"))
				if gameVer == "" {
					gameVer = strings.TrimSpace(ctx.Query("game_version"))
				}
				mmIP := strings.TrimSpace(ctx.PostForm("match_maker_ip"))
				if mmIP == "" {
					mmIP = strings.TrimSpace(ctx.Query("match_maker_ip"))
				}
				room = &rest.RoomInfo{
					LobbyCode:      lobbyCode,
					ServerIP:       serverIP,
					ServerPort:     serverPort,
					MatchMakerIp:   mmIP,
					MatchMakerPort: matchMakerPort,
					GameVersion:    gameVer,
				}
			}
		}

		rs, err := srv.UpdateSharedLobbyRoom(sessionID, hostKey, room)
		if err != nil {
			switch err {
			case service.ErrShareLobbyNotFound:
				ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			case service.ErrShareLobbyUnauthorized:
				ctx.JSON(http.StatusForbidden, gin.H{"error": "invalid host key"})
			case service.ErrShareLobbyExpired:
				ctx.JSON(http.StatusGone, gin.H{"error": "session expired"})
			default:
				slog.ErrorContext(ctx, "Failed to update shared lobby room", "error", err, "session_id", sessionID)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update lobby room"})
			}
			return
		}
		joinPath := combinePath(pathPrefix, basePath, rest.EndpointJoinLobby.Route)
		rs.URL = absoluteURL(ctx, joinPath+"?session_id="+url.QueryEscape(rs.SessionID))
		ctx.JSON(http.StatusOK, rs)
	})

	api.POST(rest.EndpointHeartbeatLobby.Route, func(ctx *gin.Context) {
		sessionID := strings.TrimSpace(ctx.Query("session_id"))
		if sessionID == "" {
			sessionID = strings.TrimSpace(ctx.PostForm("session_id"))
		}
		hostKey := strings.TrimSpace(ctx.Query("host_key"))
		if hostKey == "" {
			hostKey = strings.TrimSpace(ctx.PostForm("host_key"))
		}
		if sessionID == "" || hostKey == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id and host_key are required"})
			return
		}
		rs, err := srv.UpdateSharedLobbyExpiration(sessionID, hostKey)
		if err != nil {
			switch err {
			case service.ErrShareLobbyNotFound:
				ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			case service.ErrShareLobbyUnauthorized:
				ctx.JSON(http.StatusForbidden, gin.H{"error": "invalid host key"})
			default:
				slog.ErrorContext(ctx, "Failed to heartbeat shared lobby", "error", err, "session_id", sessionID)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to heartbeat shared lobby"})
			}
			return
		}
		joinPath := combinePath(pathPrefix, basePath, rest.EndpointJoinLobby.Route)
		rs.URL = absoluteURL(ctx, joinPath+"?session_id="+url.QueryEscape(rs.SessionID))
		ctx.JSON(http.StatusOK, rs)
	})

	api.DELETE(rest.EndpointDeleteLobby.Route, func(ctx *gin.Context) {
		sessionID := strings.TrimSpace(ctx.Param("session_id"))
		if sessionID == "" {
			sessionID = strings.TrimSpace(ctx.Query("session_id"))
		}
		hostKey := strings.TrimSpace(ctx.Query("host_key"))
		if hostKey == "" {
			hostKey = strings.TrimSpace(ctx.PostForm("host_key"))
		}
		if sessionID == "" || hostKey == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id and host_key are required"})
			return
		}
		if err := srv.DeleteSharedLobby(sessionID, hostKey); err != nil {
			switch err {
			case service.ErrShareLobbyNotFound:
				ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			case service.ErrShareLobbyUnauthorized:
				ctx.JSON(http.StatusForbidden, gin.H{"error": "invalid host key"})
			default:
				slog.ErrorContext(ctx, "Failed to delete shared lobby", "error", err, "session_id", sessionID)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete shared lobby"})
			}
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api.POST(rest.EndpointAddLobbyMember.Route, func(ctx *gin.Context) {
		sessionID := strings.TrimSpace(ctx.Param("session_id"))
		if sessionID == "" {
			sessionID = strings.TrimSpace(ctx.Query("session_id"))
		}
		if sessionID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}

		var req rest.JoinLobbyMemberRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			if uidStr := strings.TrimSpace(ctx.PostForm("discord_user_id")); uidStr != "" {
				if parsed, err := strconv.ParseUint(uidStr, 10, 64); err == nil {
					req.DiscordUserID = parsed
				}
			}
		}
		if req.DiscordUserID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "discord_user_id is required"})
			return
		}

		if err := srv.AddLobbyMember(sessionID, req.DiscordUserID); err != nil {
			switch err {
			case service.ErrShareLobbyNotFound:
				ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			case service.ErrShareLobbyExpired:
				ctx.JSON(http.StatusGone, gin.H{"error": "session expired"})
			default:
				slog.ErrorContext(ctx, "Failed to add member to lobby", "error", err, "session_id", sessionID)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add member to lobby"})
			}
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api.DELETE(rest.EndpointRemoveLobbyMember.Route, func(ctx *gin.Context) {
		sessionID := strings.TrimSpace(ctx.Param("session_id"))
		userIDStr := strings.TrimSpace(ctx.Param("user_id"))
		if sessionID == "" || userIDStr == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id and user_id are required"})
			return
		}
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil || userID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}

		if err := srv.RemoveLobbyMember(sessionID, userID); err != nil {
			switch err {
			case service.ErrShareLobbyNotFound:
				ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			case service.ErrShareLobbyExpired:
				ctx.JSON(http.StatusGone, gin.H{"error": "session expired"})
			default:
				slog.ErrorContext(ctx, "Failed to remove member from lobby", "error", err, "session_id", sessionID, "user_id", userID)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove member from lobby"})
			}
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api.GET(rest.EndpointJoinLobby.Route, func(ctx *gin.Context) {
		sessionID := strings.TrimSpace(ctx.Query("session_id"))
		if sessionID == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}
		if ctx.Query("download") != "" {
			data, err := srv.GetJoinLobbyDownload(sessionID)
			if err != nil {
				switch err {
				case service.ErrShareLobbyNotFound:
					ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
				case service.ErrShareLobbyExpired:
					ctx.JSON(http.StatusGone, gin.H{"error": "session expired"})
				default:
					slog.ErrorContext(ctx, "Failed to get join lobby download", "error", err, "session_id", sessionID)
					ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get join lobby data"})
				}
				return
			}
			ctx.JSON(http.StatusOK, data)
			return
		}

		serverBase := absoluteURL(ctx, combinePath(pathPrefix, basePath, ""))
		_, err := srv.GetJoinLobbyMeta(sessionID)
		if err != nil {
			message := "このロビー参加リンクは無効です。時間切れの可能性があります。"
			errorType := rest.JoinLobbyErrorInvalidSession
			switch err {
			case service.ErrShareLobbyExpired:
				message = "このロビー参加リンクは有効期限切れです。"
				errorType = rest.JoinLobbyErrorSessionExpired
			case service.ErrShareLobbyNotFound:
				message = "このロビー参加リンクは見つかりません。"
				errorType = rest.JoinLobbyErrorSessionNotFound
			}
			deepLink := buildJoinLobbyDeepLink(serverBase, sessionID, errorType)
			ctx.Header("Content-Type", "text/html; charset=utf-8")
			ctx.String(http.StatusNotFound, joinGameHTML(message, deepLink, false))
			return
		}
		deepLink := buildJoinLobbyDeepLink(serverBase, sessionID, "")
		ctx.Header("Content-Type", "text/html; charset=utf-8")
		ctx.String(http.StatusOK, joinGameHTML("", deepLink, true))
	})

	if pathPrefix != "" && pathPrefix != "/" {
		return http.StripPrefix(pathPrefix, r.Handler())
	}

	return r.Handler()
}

func clientIP(ctx *gin.Context) string {
	forwarded := strings.TrimSpace(ctx.GetHeader("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			v := strings.TrimSpace(parts[0])
			if v != "" {
				return v
			}
		}
	}
	return ctx.ClientIP()
}

func combinePath(parts ...string) string {
	buf := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "/" {
			continue
		}
		buf = append(buf, strings.Trim(p, "/"))
	}
	if len(buf) == 0 {
		return "/"
	}
	return "/" + strings.Join(buf, "/")
}

func absoluteURL(ctx *gin.Context, path string) string {
	scheme := "http"
	if proto := strings.TrimSpace(ctx.GetHeader("X-Forwarded-Proto")); proto != "" {
		scheme = proto
	} else if ctx.Request.TLS != nil {
		scheme = "https"
	}
	host := ctx.Request.Host
	return fmt.Sprintf("%s://%s%s", scheme, host, path)
}

func buildJoinGameDeepLink(serverBase, sessionID, errorType string) string {
	values := make(url.Values)
	values.Set("server", serverBase)
	if errorType != "" {
		values.Set("error_type", errorType)
	}
	return "mod-of-us://join_game/v1/" + url.PathEscape(sessionID) + "?" + values.Encode()
}

func buildJoinLobbyDeepLink(serverBase, sessionID, errorType string) string {
	values := make(url.Values)
	values.Set("server", serverBase)
	if errorType != "" {
		values.Set("error_type", errorType)
	}
	return "mod-of-us://join_lobby/v1/" + url.PathEscape(sessionID) + "?" + values.Encode()
}

//go:embed templates/join_game.tmpl
var joinGameTemplateFS embed.FS

var joinGameTemplate = template.Must(template.ParseFS(joinGameTemplateFS, "templates/join_game.tmpl"))

const launcherReleaseURL = "https://github.com/ikafly144/au_mod_installer/releases/latest"

type joinGameData struct {
	Status     string
	Message    string
	DeepLink   template.URL
	ReleaseURL template.URL
}

func joinGameHTML(message, deepLink string, success bool) string {
	status := "参加リンクを開いています..."
	if !success {
		status = "参加リンクを開けませんでした。"
	}
	data := joinGameData{
		Status:     status,
		Message:    message,
		DeepLink:   template.URL(deepLink),
		ReleaseURL: template.URL(launcherReleaseURL),
	}
	var buf bytes.Buffer
	if err := joinGameTemplate.Execute(&buf, data); err != nil {
		slog.Error("failed to execute join game template", "error", err)
		return ""
	}
	return buf.String()
}
