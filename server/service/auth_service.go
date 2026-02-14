package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ikafly144/au_mod_installer/server/model"
	"github.com/ikafly144/au_mod_installer/server/repository"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrOAuthFailed        = errors.New("discord oauth failed")
)

type AuthService struct {
	userRepo            repository.UserRepository
	jwtSecret           []byte
	discordClientID     string
	discordClientSecret string
	discordRedirectURI  string
	adminDiscordIDs     map[string]bool
	httpClient          *http.Client
}

type AuthServiceConfig struct {
	UserRepo            repository.UserRepository
	JWTSecret           string
	DiscordClientID     string
	DiscordClientSecret string
	DiscordRedirectURI  string
	AdminDiscordIDs     []string
}

func NewAuthService(config AuthServiceConfig) *AuthService {
	adminIDs := make(map[string]bool)
	for _, id := range config.AdminDiscordIDs {
		adminIDs[id] = true
	}
	return &AuthService{
		userRepo:            config.UserRepo,
		jwtSecret:           []byte(config.JWTSecret),
		discordClientID:     config.DiscordClientID,
		discordClientSecret: config.DiscordClientSecret,
		discordRedirectURI:  config.DiscordRedirectURI,
		adminDiscordIDs:     adminIDs,
		httpClient:          &http.Client{Timeout: 10 * time.Second},
	}
}

// SetHTTPClient allows overriding the HTTP client (for testing)
func (s *AuthService) SetHTTPClient(client *http.Client) {
	s.httpClient = client
}

// GetDiscordAuthURL returns the Discord OAuth2 authorization URL
func (s *AuthService) GetDiscordAuthURL() string {
	params := url.Values{
		"client_id":     {s.discordClientID},
		"redirect_uri":  {s.discordRedirectURI},
		"response_type": {"code"},
		"scope":         {"identify"},
	}
	return "https://discord.com/api/oauth2/authorize?" + params.Encode()
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

// discordTokenResponse is the response from Discord's token endpoint
type discordTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// discordUser is the response from Discord's /users/@me endpoint
type discordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	GlobalName    string `json:"global_name"`
	Avatar        string `json:"avatar"`
	Discriminator string `json:"discriminator"`
}

// DiscordOAuthLogin exchanges a Discord OAuth code for user info and returns a JWT
func (s *AuthService) DiscordOAuthLogin(ctx context.Context, code string) (*LoginResponse, error) {
	// 1. Exchange code for access token
	tokenResp, err := s.exchangeCode(code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthFailed, err)
	}

	// 2. Fetch Discord user info
	discordUser, err := s.fetchDiscordUser(tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthFailed, err)
	}

	// 3. Upsert user in database
	user, err := s.upsertUser(ctx, discordUser)
	if err != nil {
		return nil, err
	}

	// 4. Generate JWT
	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) exchangeCode(code string) (*discordTokenResponse, error) {
	data := url.Values{
		"client_id":     {s.discordClientID},
		"client_secret": {s.discordClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.discordRedirectURI},
	}

	req, err := http.NewRequest("POST", "https://discord.com/api/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp discordTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (s *AuthService) fetchDiscordUser(accessToken string) (*discordUser, error) {
	req, err := http.NewRequest("GET", "https://discord.com/api/v10/users/@me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord user fetch failed with status %d", resp.StatusCode)
	}

	var user discordUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *AuthService) upsertUser(ctx context.Context, du *discordUser) (*model.User, error) {
	displayName := du.GlobalName
	if displayName == "" {
		displayName = du.Username
	}

	avatarURL := ""
	if du.Avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", du.ID, du.Avatar)
	}

	existing, err := s.userRepo.GetUserByDiscordID(ctx, du.ID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		// Update existing user info from Discord
		existing.Username = du.Username
		existing.DisplayName = displayName
		existing.AvatarURL = avatarURL
		// Grant admin if in admin list (don't revoke existing admin)
		if s.adminDiscordIDs[du.ID] {
			existing.IsAdmin = true
		}
		if err := s.userRepo.UpdateUser(ctx, *existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	// Create new user
	newUser := model.User{
		DiscordID:   du.ID,
		Username:    du.Username,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		IsAdmin:     s.adminDiscordIDs[du.ID],
	}

	if err := s.userRepo.CreateUser(ctx, newUser); err != nil {
		return nil, err
	}

	// Fetch the created user to get the ID
	created, err := s.userRepo.GetUserByDiscordID(ctx, du.ID)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *AuthService) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":        user.ID,
		"discord_id": user.DiscordID,
		"username":   user.Username,
		"is_admin":   user.IsAdmin,
		"exp":        time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
