package main

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/server/service"
)

func router(srv *service.ModService) http.Handler {
	r := gin.Default()

	r.GET(rest.EndpointGetModList.Route, func(ctx *gin.Context) {
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

		ctx.JSON(http.StatusOK, gin.H{
			"ids":     modIDs,
			"next_id": nextID,
		})
	})
	r.GET(rest.EndpointGetModDetail.Route, func(ctx *gin.Context) {
		modID := ctx.Param("mod_id")

		details, err := srv.GetModDetails(modID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get mod details", "mod_id", modID, "error", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get mod details"})
			return
		}

		ctx.JSON(http.StatusOK, details)
	})
	r.GET(rest.EndpointGetModVersionList.Route, func(ctx *gin.Context) {
		modID := ctx.Param("mod_id")

		versionIDs, err := srv.GetModVersionIds(modID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get mod version IDs", "mod_id", modID, "error", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get mod version IDs"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"ids": versionIDs,
		})
	})
	r.GET(rest.EndpointGetModVersionDetail.Route, func(ctx *gin.Context) {
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
	r.GET(rest.EndpointHealth.Route, func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r.Handler()
}
