package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ikafly144/au_mod_installer/server/model"
	"github.com/ikafly144/au_mod_installer/server/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) GetUser(ctx context.Context, id int) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepo) GetUserByDiscordID(ctx context.Context, discordID string) (*model.User, error) {
	args := m.Called(ctx, discordID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepo) CreateUser(ctx context.Context, user model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepo) UpdateUser(ctx context.Context, user model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepo) DeleteUser(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

var _ repository.UserRepository = (*MockUserRepo)(nil)

func TestAuthService_DiscordOAuthLogin_NewUser(t *testing.T) {
	// Mock Discord API server
	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/oauth2/token"):
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock_token",
				"token_type":   "Bearer",
			})
		case strings.HasSuffix(r.URL.Path, "/users/@me"):
			json.NewEncoder(w).Encode(map[string]string{
				"id":            "123456789",
				"username":      "testuser",
				"global_name":   "Test User",
				"avatar":        "abc123",
				"discriminator": "0",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer discordServer.Close()

	mockRepo := new(MockUserRepo)
	svc := NewAuthService(AuthServiceConfig{
		UserRepo:            mockRepo,
		JWTSecret:           "secret",
		DiscordClientID:     "client_id",
		DiscordClientSecret: "client_secret",
		DiscordRedirectURI:  "http://localhost/callback",
	})

	// Override the Discord API base URLs via custom transport that redirects to our mock server
	transport := &redirectTransport{
		targetBase: discordServer.URL,
		inner:      http.DefaultTransport,
	}
	svc.SetHTTPClient(&http.Client{Transport: transport})

	ctx := context.Background()

	// First call: user does not exist → create
	mockRepo.On("GetUserByDiscordID", mock.Anything, "123456789").Return(nil, nil).Once()
	mockRepo.On("CreateUser", mock.Anything, mock.MatchedBy(func(u model.User) bool {
		return u.DiscordID == "123456789" && u.Username == "testuser"
	})).Return(nil)
	mockRepo.On("GetUserByDiscordID", mock.Anything, "123456789").Return(&model.User{
		ID:          1,
		DiscordID:   "123456789",
		Username:    "testuser",
		DisplayName: "Test User",
		AvatarURL:   "https://cdn.discordapp.com/avatars/123456789/abc123.png",
	}, nil).Once()

	resp, err := svc.DiscordOAuthLogin(ctx, "mock_code")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "testuser", resp.User.Username)
	assert.Equal(t, "123456789", resp.User.DiscordID)

	mockRepo.AssertExpectations(t)
}

func TestAuthService_DiscordOAuthLogin_ExistingUser(t *testing.T) {
	// Mock Discord API server
	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/oauth2/token"):
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "mock_token",
				"token_type":   "Bearer",
			})
		case strings.HasSuffix(r.URL.Path, "/users/@me"):
			json.NewEncoder(w).Encode(map[string]string{
				"id":            "123456789",
				"username":      "testuser_updated",
				"global_name":   "Updated Name",
				"avatar":        "def456",
				"discriminator": "0",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer discordServer.Close()

	mockRepo := new(MockUserRepo)
	svc := NewAuthService(AuthServiceConfig{
		UserRepo:            mockRepo,
		JWTSecret:           "secret",
		DiscordClientID:     "client_id",
		DiscordClientSecret: "client_secret",
		DiscordRedirectURI:  "http://localhost/callback",
	})
	svc.SetHTTPClient(&http.Client{Transport: &redirectTransport{
		targetBase: discordServer.URL,
		inner:      http.DefaultTransport,
	}})

	ctx := context.Background()

	existingUser := &model.User{
		ID:          1,
		DiscordID:   "123456789",
		Username:    "testuser",
		DisplayName: "Test User",
		IsAdmin:     true,
	}

	mockRepo.On("GetUserByDiscordID", mock.Anything, "123456789").Return(existingUser, nil)
	mockRepo.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u model.User) bool {
		return u.Username == "testuser_updated" && u.DisplayName == "Updated Name" && u.IsAdmin == true
	})).Return(nil)

	resp, err := svc.DiscordOAuthLogin(ctx, "mock_code")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "testuser_updated", resp.User.Username)
	assert.True(t, resp.User.IsAdmin) // Admin status preserved

	mockRepo.AssertExpectations(t)
}

// redirectTransport redirects all requests to a target base URL (the mock server)
// while preserving the original path
type redirectTransport struct {
	targetBase string
	inner      http.RoundTripper
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect to mock server while preserving path
	newURL := t.targetBase + req.URL.Path
	if req.URL.RawQuery != "" {
		newURL += "?" + req.URL.RawQuery
	}
	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header
	return t.inner.RoundTrip(newReq)
}
