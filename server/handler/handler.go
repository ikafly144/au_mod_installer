package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/server/service"
)

// ModServiceInterface defines the interface for mod service operations
type ModServiceInterface interface {
	GetModList(ctx context.Context, limit int, after string, before string) ([]modmgr.Mod, error)
	GetMod(ctx context.Context, modID string) (*modmgr.Mod, error)
	GetModVersions(ctx context.Context, modID string, limit int, after string) ([]modmgr.ModVersion, error)
	GetModVersion(ctx context.Context, modID string, versionID string) (*modmgr.ModVersion, error)
}

type Handler struct {
	modService ModServiceInterface
}

func NewHandler(modService ModServiceInterface) *Handler {
	return &Handler{
		modService: modService,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /mods", h.handleGetMods)
	mux.HandleFunc("GET /mods/{modID}", h.handleGetMod)
	mux.HandleFunc("GET /mods/{modID}/versions", h.handleGetModVersions)
	mux.HandleFunc("GET /mods/{modID}/versions/{versionID}", h.handleGetModVersion)
}

func (h *Handler) handleGetMods(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit := 0
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	after := query.Get("after")
	before := query.Get("before")

	mods, err := h.modService.GetModList(r.Context(), limit, after, before)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "mods not found")
			return
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, mods)
}

func (h *Handler) handleGetMod(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		writeError(w, http.StatusBadRequest, "modID is required")
		return
	}

	mod, err := h.modService.GetMod(r.Context(), modID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "mods not found")
			return
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if mod == nil {
		writeError(w, http.StatusNotFound, "mod not found")
		return
	}

	writeJSON(w, http.StatusOK, mod)
}

func (h *Handler) handleGetModVersions(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		writeError(w, http.StatusBadRequest, "modID is required")
		return
	}

	query := r.URL.Query()

	limit := 0
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	after := query.Get("after")

	versions, err := h.modService.GetModVersions(r.Context(), modID, limit, after)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "mods not found")
			return
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, versions)
}

func (h *Handler) handleGetModVersion(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	versionID := r.PathValue("versionID")

	if modID == "" {
		writeError(w, http.StatusBadRequest, "modID is required")
		return
	}
	if versionID == "" {
		writeError(w, http.StatusBadRequest, "versionID is required")
		return
	}

	version, err := h.modService.GetModVersion(r.Context(), modID, versionID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "version not found")
			return
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if version == nil {
		writeError(w, http.StatusNotFound, "version not found")
		return
	}

	writeJSON(w, http.StatusOK, version)
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// extractPathParams extracts path parameters from URL pattern matching
// e.g., for pattern "/mods/{modID}/versions/{versionID}" and path "/mods/123/versions/456"
// returns map[string]string{"modID": "123", "versionID": "456"}
func extractPathParams(pattern, path string) map[string]string {
	params := make(map[string]string)

	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return params
	}

	for i, part := range patternParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			paramName := part[1 : len(part)-1]
			params[paramName] = pathParts[i]
		}
	}

	return params
}
