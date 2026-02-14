package handler

import (
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

	mux.HandleFunc("GET "+p("/auth/discord"), h.handleDiscordRedirect)
	mux.HandleFunc("GET "+p("/auth/discord/callback"), h.handleDiscordCallback)
}

func (h *AuthHandler) handleDiscordRedirect(w http.ResponseWriter, r *http.Request) {
	url := h.authService.GetDiscordAuthURL()
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) handleDiscordCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		WriteError(w, http.StatusBadRequest, "missing code parameter")
		return
	}

	resp, err := h.authService.DiscordOAuthLogin(r.Context(), code)
	if err != nil {
		if err == service.ErrOAuthFailed {
			WriteError(w, http.StatusUnauthorized, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}
