package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ikafly144/au_mod_installer/server/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux, basePath string) {
	p := func(path string) string {
		if basePath == "" {
			return path
		}
		return basePath + path
	}

	mux.HandleFunc("POST "+p("/auth/register"), h.handleRegister)
	mux.HandleFunc("POST "+p("/auth/login"), h.handleLogin)
}

func (h *AuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.authService.Register(r.Context(), req)
	if err != nil {
		if err == service.ErrUsernameTaken {
			WriteError(w, http.StatusConflict, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req service.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authService.Login(r.Context(), req)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			WriteError(w, http.StatusUnauthorized, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}
