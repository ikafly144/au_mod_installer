package main

import (
	"fmt"
	"html"
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

func router(srv *service.ModService, pathPrefix string, basePath string) http.Handler {
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
	api.POST(rest.EndpointShareGame.Route, func(ctx *gin.Context) {
		var req rest.ShareGameRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
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
			message := "この部屋リンクは無効です。時間切れの可能性があります。"
			switch err {
			case service.ErrShareGameExpired:
				message = "この部屋リンクは有効期限切れです。"
			case service.ErrShareGameNotFound:
				message = "この部屋リンクは見つかりません。"
			}
			deepLink := buildJoinGameDeepLink(serverBase, sessionID, message)
			ctx.Header("Content-Type", "text/html; charset=utf-8")
			ctx.String(http.StatusNotFound, joinGameHTML(message, deepLink, false))
			return
		}
		deepLink := buildJoinGameDeepLink(serverBase, sessionID, "")
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

func buildJoinGameDeepLink(serverBase, sessionID, errorMessage string) string {
	values := make(url.Values)
	values.Set("server", serverBase)
	if errorMessage != "" {
		values.Set("error", errorMessage)
	}
	return "mod-of-us://join_game/v1/" + url.PathEscape(sessionID) + "?" + values.Encode()
}

func joinGameHTML(message, deepLink string, success bool) string {
	status := "部屋リンクを開いています..."
	if !success {
		status = "部屋リンクを開けませんでした。"
	}
	messageHTML := ""
	if message != "" {
		messageHTML = `<p class="error">` + html.EscapeString(message) + `</p>`
	}
	return `<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Mod of Us 部屋リンク</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#111;color:#eee;padding:24px;line-height:1.5}
.card{max-width:560px;margin:0 auto;background:#1b1b1b;border:1px solid #2f2f2f;border-radius:10px;padding:20px}
.error{color:#ff8b8b}
a.btn{display:inline-block;padding:10px 14px;background:#2d7ef7;color:#fff;text-decoration:none;border-radius:8px}
</style>
</head>
<body>
<div class="card">
<h1>Mod of Us 部屋リンク</h1>
<p>` + html.EscapeString(status) + `</p>
` + messageHTML + `
<p><a class="btn" href="` + html.EscapeString(deepLink) + `">ランチャーで開く</a></p>
<p>自動で開かない場合は上のボタンを押してください。</p>
</div>
<script>
(()=>{const link="` + deepLink + `";
try{window.location.href=link;}catch(e){}
})();
</script>
</body>
</html>`
}
