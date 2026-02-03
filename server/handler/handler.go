package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"strings"

	"github.com/ikafly144/au_mod_installer/common/rest"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/server/middleware"
	"github.com/ikafly144/au_mod_installer/server/service"
)

// ModServiceInterface defines the interface for mod service operations
const (
	DefaultLimit = 50
	MaxLimit     = 100
)

type ModServiceInterface interface {
	GetModList(ctx context.Context, limit int, after string, before string) ([]modmgr.Mod, error)
	GetMod(ctx context.Context, modID string) (*modmgr.Mod, error)
	GetModVersions(ctx context.Context, modID string, limit int, after string) ([]modmgr.ModVersion, error)
	GetModVersion(ctx context.Context, modID string, versionID string) (*modmgr.ModVersion, error)

	CreateMod(ctx context.Context, mod modmgr.Mod) error
	UpdateMod(ctx context.Context, mod modmgr.Mod) error
	DeleteMod(ctx context.Context, modID string) error
	CreateModVersion(ctx context.Context, modID string, version modmgr.ModVersion) error
	DeleteModVersion(ctx context.Context, modID string, versionID string) error
}

type Handler struct {
	modService       ModServiceInterface
	version          string
	disabledVersions []string
	authMiddleware   *middleware.AuthMiddleware
}

func NewHandler(modService ModServiceInterface, version string, disabledVersions []string) *Handler {
	return &Handler{
		modService:       modService,
		version:          version,
		disabledVersions: disabledVersions,
	}
}

func (h *Handler) SetAuthMiddleware(mw *middleware.AuthMiddleware) {
	h.authMiddleware = mw
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, basePath string) {
	// helper to prepend base path
	p := func(pattern string) string {
		if basePath == "" {
			return pattern
		}

		method, path, found := strings.Cut(pattern, " ")
		if !found {
			path = pattern
			method = ""
		}

		// Ensure basePath doesn't have trailing slash
		cleanedBase := strings.TrimRight(basePath, "/")
		if cleanedBase == "" && strings.HasPrefix(basePath, "/") {
			// basePath was just "/" or "///"
			cleanedBase = ""
		}

		newPath := cleanedBase + path
		if method != "" {
			return method + " " + newPath
		}
		return newPath
	}

	secure := func(next http.HandlerFunc) http.Handler {
		if h.authMiddleware == nil {
			// If no auth middleware is configured (e.g. no secret), we might block write operations
			// or allow them if that's the policy. For now, let's block them.
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				WriteError(w, http.StatusServiceUnavailable, "authentication not configured")
			})
		}
		return h.authMiddleware.Middleware(next)
	}

	mux.HandleFunc(p("GET /health"), h.handleHealth)
	mux.HandleFunc(p("GET /mods"), h.handleGetMods)
	mux.HandleFunc(p("GET /mods/{modID}"), h.handleGetMod)
	mux.HandleFunc(p("GET /mods/{modID}/versions"), h.handleGetModVersions)
	mux.HandleFunc(p("GET /mods/{modID}/versions/{versionID}"), h.handleGetModVersion)

	mux.Handle(p("POST /mods"), secure(h.handleCreateMod))
	mux.Handle(p("PUT /mods/{modID}"), secure(h.handleUpdateMod))
	mux.Handle(p("DELETE /mods/{modID}"), secure(h.handleDeleteMod))
	mux.Handle(p("POST /mods/{modID}/versions"), secure(h.handleCreateModVersion))
	mux.Handle(p("DELETE /mods/{modID}/versions/{versionID}"), secure(h.handleDeleteModVersion))
	mux.Handle(p("POST /upload"), secure(h.handleUpload))
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {

	WriteJSON(w, http.StatusOK, rest.HealthStatus{
		Status:           "OK",
		WorkingVersion:   h.version,
		DisabledVersions: h.disabledVersions,
	})
}

func (h *Handler) handleGetMods(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit := DefaultLimit
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	after := query.Get("after")
	before := query.Get("before")

	mods, err := h.modService.GetModList(r.Context(), limit, after, before)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "mods not found")
			return
		} else {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	WriteJSON(w, http.StatusOK, mods)
}

func (h *Handler) handleGetMod(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		WriteError(w, http.StatusBadRequest, "modID is required")
		return
	}

	mod, err := h.modService.GetMod(r.Context(), modID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "mods not found")
			return
		} else {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if mod == nil {
		WriteError(w, http.StatusNotFound, "mod not found")
		return
	}

	WriteJSON(w, http.StatusOK, mod)
}

func (h *Handler) handleGetModVersions(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		WriteError(w, http.StatusBadRequest, "modID is required")
		return
	}

	query := r.URL.Query()

	limit := DefaultLimit
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	after := query.Get("after")

	versions, err := h.modService.GetModVersions(r.Context(), modID, limit, after)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "mods not found")
			return
		} else {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	WriteJSON(w, http.StatusOK, versions)
}

func (h *Handler) handleGetModVersion(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	versionID := r.PathValue("versionID")

	if modID == "" {
		WriteError(w, http.StatusBadRequest, "modID is required")
		return
	}
	if versionID == "" {
		WriteError(w, http.StatusBadRequest, "versionID is required")
		return
	}

	version, err := h.modService.GetModVersion(r.Context(), modID, versionID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "version not found")
			return
		} else {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if version == nil {
		WriteError(w, http.StatusNotFound, "version not found")
		return
	}

	WriteJSON(w, http.StatusOK, version)
}

func (h *Handler) handleCreateMod(w http.ResponseWriter, r *http.Request) {
	var mod modmgr.Mod
	if err := json.NewDecoder(r.Body).Decode(&mod); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if mod.ID == "" || mod.Name == "" {
		WriteError(w, http.StatusBadRequest, "id and name are required")
		return
	}

	if err := h.modService.CreateMod(r.Context(), mod); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, mod)
}

func (h *Handler) handleUpdateMod(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		WriteError(w, http.StatusBadRequest, "modID is required")
		return
	}

	var mod modmgr.Mod
	if err := json.NewDecoder(r.Body).Decode(&mod); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if mod.ID != "" && mod.ID != modID {
		WriteError(w, http.StatusBadRequest, "modID in path and body must match")
		return
	}
	mod.ID = modID

	if err := h.modService.UpdateMod(r.Context(), mod); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, mod)
}

func (h *Handler) handleDeleteMod(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		WriteError(w, http.StatusBadRequest, "modID is required")
		return
	}

	if err := h.modService.DeleteMod(r.Context(), modID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleCreateModVersion(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		WriteError(w, http.StatusBadRequest, "modID is required")
		return
	}

	var version modmgr.ModVersion
	if err := json.NewDecoder(r.Body).Decode(&version); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	version.ModID = modID
	if version.ID == "" {
		WriteError(w, http.StatusBadRequest, "version id is required")
		return
	}

	if err := h.modService.CreateModVersion(r.Context(), modID, version); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, version)
}

func (h *Handler) handleDeleteModVersion(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	versionID := r.PathValue("versionID")

	if modID == "" || versionID == "" {
		WriteError(w, http.StatusBadRequest, "modID and versionID are required")
		return
	}

	if err := h.modService.DeleteModVersion(r.Context(), modID, versionID); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleUpload(w http.ResponseWriter, r *http.Request) {
	// Limit 50MB
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		WriteError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")

	if err != nil {
		WriteError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to create upload dir")
		return
	}

	// Simple filename sanitization
	filename := filepath.Base(header.Filename)
	filePath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		WriteError(w, http.StatusInternalServerError, "failed to copy file content")
		return
	}

	// Generate URL (relative or absolute)
	// Assuming /uploads is served at /uploads
	url := "/uploads/" + filename
	WriteJSON(w, http.StatusCreated, map[string]string{"url": url})
}

type errorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, errorResponse{Error: message})
}
