package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ikafly144/au_mod_installer/server-admin/model"
	"github.com/ikafly144/au_mod_installer/server-admin/repository"
	"github.com/ikafly144/au_mod_installer/server-admin/templates"
)

// Handler handles HTTP requests
type Handler struct {
	repo repository.Repository
	tmpl *templates.Templates
}

// New creates a new Handler
func New(repo repository.Repository, tmpl *templates.Templates) *Handler {
	return &Handler{repo: repo, tmpl: tmpl}
}

// HandleIndex renders the mod list page
func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	mods, err := h.repo.GetModList(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title": "Mod一覧",
		"Mods":  mods,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.Render(w, "index", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HandleVersionsPage renders the version list page
func (h *Handler) HandleVersionsPage(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		http.Error(w, "modID is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	mod, err := h.repo.GetMod(ctx, modID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if mod == nil {
		http.Error(w, "mod not found", http.StatusNotFound)
		return
	}

	versions, err := h.repo.GetVersionList(ctx, modID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":    fmt.Sprintf("バージョン - %s", mod.Name),
		"Mod":      mod,
		"Versions": versions,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.Render(w, "versions", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HandleGetMods returns all mods as JSON
func (h *Handler) HandleGetMods(w http.ResponseWriter, r *http.Request) {
	mods, err := h.repo.GetModList(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, mods)
}

// HandleGetMod returns a specific mod
func (h *Handler) HandleGetMod(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		writeError(w, http.StatusBadRequest, "modID is required")
		return
	}

	mod, err := h.repo.GetMod(r.Context(), modID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if mod == nil {
		writeError(w, http.StatusNotFound, "mod not found")
		return
	}

	writeJSON(w, http.StatusOK, mod)
}

// HandleCreateMod creates a new mod
func (h *Handler) HandleCreateMod(w http.ResponseWriter, r *http.Request) {
	var mod model.Mod
	if err := json.NewDecoder(r.Body).Decode(&mod); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if mod.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	ctx := r.Context()

	// Check if mod already exists
	existing, err := h.repo.GetMod(ctx, mod.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "mod already exists")
		return
	}

	if err := h.repo.SetMod(ctx, mod); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, mod)
}

// HandleUpdateMod updates an existing mod
func (h *Handler) HandleUpdateMod(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		writeError(w, http.StatusBadRequest, "modID is required")
		return
	}

	var mod model.Mod
	if err := json.NewDecoder(r.Body).Decode(&mod); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	mod.ID = modID
	ctx := r.Context()

	// Check if mod exists
	existing, err := h.repo.GetMod(ctx, modID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "mod not found")
		return
	}

	if err := h.repo.SetMod(ctx, mod); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, mod)
}

// HandleDeleteMod deletes a mod
func (h *Handler) HandleDeleteMod(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		writeError(w, http.StatusBadRequest, "modID is required")
		return
	}

	if err := h.repo.DeleteMod(r.Context(), modID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleGetVersions returns all versions of a mod
func (h *Handler) HandleGetVersions(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		writeError(w, http.StatusBadRequest, "modID is required")
		return
	}

	versions, err := h.repo.GetVersionList(r.Context(), modID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, versions)
}

// HandleGetVersion returns a specific version
func (h *Handler) HandleGetVersion(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	versionID := r.PathValue("versionID")

	if modID == "" || versionID == "" {
		writeError(w, http.StatusBadRequest, "modID and versionID are required")
		return
	}

	version, err := h.repo.GetVersion(r.Context(), modID, versionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if version == nil {
		writeError(w, http.StatusNotFound, "version not found")
		return
	}

	writeJSON(w, http.StatusOK, version)
}

// HandleCreateVersion creates a new version
func (h *Handler) HandleCreateVersion(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	if modID == "" {
		writeError(w, http.StatusBadRequest, "modID is required")
		return
	}

	var version model.ModVersion
	if err := json.NewDecoder(r.Body).Decode(&version); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if version.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	ctx := r.Context()

	// Check if mod exists
	mod, err := h.repo.GetMod(ctx, modID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if mod == nil {
		writeError(w, http.StatusNotFound, "mod not found")
		return
	}

	if version.CreatedAt.IsZero() {
		version.CreatedAt = time.Now()
	}

	if err := h.repo.SetVersion(ctx, modID, version); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, version)
}

// HandleUpdateVersion updates an existing version
func (h *Handler) HandleUpdateVersion(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	versionID := r.PathValue("versionID")

	if modID == "" || versionID == "" {
		writeError(w, http.StatusBadRequest, "modID and versionID are required")
		return
	}

	var version model.ModVersion
	if err := json.NewDecoder(r.Body).Decode(&version); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	version.ID = versionID
	ctx := r.Context()

	// Check if version exists
	existing, err := h.repo.GetVersion(ctx, modID, versionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "version not found")
		return
	}

	if version.CreatedAt.IsZero() {
		version.CreatedAt = existing.CreatedAt
	}

	if err := h.repo.SetVersion(ctx, modID, version); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, version)
}

// HandleDeleteVersion deletes a version
func (h *Handler) HandleDeleteVersion(w http.ResponseWriter, r *http.Request) {
	modID := r.PathValue("modID")
	versionID := r.PathValue("versionID")

	if modID == "" || versionID == "" {
		writeError(w, http.StatusBadRequest, "modID and versionID are required")
		return
	}

	if err := h.repo.DeleteVersion(r.Context(), modID, versionID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleImport imports mods from JSON
func (h *Handler) HandleImport(w http.ResponseWriter, r *http.Request) {
	var mods []model.ModWithVersions
	if err := json.NewDecoder(r.Body).Decode(&mods); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()
	imported := 0

	for _, m := range mods {
		if err := h.repo.SetMod(ctx, m.Mod); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		for _, v := range m.Versions {
			if err := h.repo.SetVersion(ctx, m.ID, v); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		imported++
	}

	writeJSON(w, http.StatusOK, map[string]int{"imported": imported})
}

// HandleExport exports all mods to JSON
func (h *Handler) HandleExport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	mods, err := h.repo.GetModList(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]model.ModWithVersions, 0, len(mods))
	for _, mod := range mods {
		versions, err := h.repo.GetVersionList(ctx, mod.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		result = append(result, model.ModWithVersions{
			Mod:      mod,
			Versions: versions,
		})
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=mods_%s.json", time.Now().Format("20060102_150405")))
	writeJSON(w, http.StatusOK, result)
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
