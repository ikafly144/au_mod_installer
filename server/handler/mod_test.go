package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ikafly144/au_mod_installer/pkg/aumgr"
	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/server/middleware"
	"github.com/ikafly144/au_mod_installer/server/service" // Import the service package to access GitHubRelease and GitHubAsset
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockModService definition (unchanged)
type MockModService struct {
	mock.Mock
}

func (m *MockModService) GetModList(ctx context.Context, limit int, after string, before string) ([]modmgr.Mod, error) {
	args := m.Called(ctx, limit, after, before)
	return args.Get(0).([]modmgr.Mod), args.Error(1)
}

func (m *MockModService) GetMod(ctx context.Context, modID string) (*modmgr.Mod, error) {
	args := m.Called(ctx, modID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*modmgr.Mod), args.Error(1)
}

func (m *MockModService) GetModVersions(ctx context.Context, modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	args := m.Called(ctx, modID, limit, after)
	return args.Get(0).([]modmgr.ModVersion), args.Error(1)
}

func (m *MockModService) GetModVersion(ctx context.Context, modID string, versionID string) (*modmgr.ModVersion, error) {
	args := m.Called(ctx, modID, versionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*modmgr.ModVersion), args.Error(1)
}

func (m *MockModService) CreateMod(ctx context.Context, mod modmgr.Mod) error {
	args := m.Called(ctx, mod)
	return args.Error(0)
}

func (m *MockModService) UpdateMod(ctx context.Context, mod modmgr.Mod) error {
	args := m.Called(ctx, mod)
	return args.Error(0)
}

func (m *MockModService) UpdateModVersion(ctx context.Context, modID string, version modmgr.ModVersion) error {
	args := m.Called(ctx, modID, version)
	return args.Error(0)
}

func (m *MockModService) DeleteMod(ctx context.Context, modID string) error {
	args := m.Called(ctx, modID)
	return args.Error(0)
}

func (m *MockModService) CreateModVersion(ctx context.Context, modID string, version modmgr.ModVersion) error {
	args := m.Called(ctx, modID, version)
	return args.Error(0)
}

func (m *MockModService) DeleteModVersion(ctx context.Context, modID string, versionID string) error {
	args := m.Called(ctx, modID, versionID)
	return args.Error(0)
}

// MockGitHubService definition
type MockGitHubService struct {
	mock.Mock
}

func (m *MockGitHubService) ListReleases(owner, repo string) ([]service.GitHubRelease, error) {
	args := m.Called(owner, repo)
	return args.Get(0).([]service.GitHubRelease), args.Error(1)
}

func (m *MockGitHubService) GetRelease(owner, repo, tag string) (*service.GitHubRelease, error) {
	args := m.Called(owner, repo, tag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.GitHubRelease), args.Error(1)
}


func TestHandler_CreateMod(t *testing.T) {
	mockSvc := new(MockModService)
	h := NewHandler(mockSvc, new(MockGitHubService), "v1.0.0", nil)

	// Setup middleware
	mw := middleware.NewAuthMiddleware("secret")
	h.SetAuthMiddleware(mw)

	// Mock CreateMod success
	mockSvc.On("CreateMod", mock.Anything, mock.MatchedBy(func(m modmgr.Mod) bool {
		return m.ID == "test-mod" && m.Name == "Test Mod"
	})).Return(nil)

	// Request with valid token
	mod := modmgr.Mod{ID: "test-mod", Name: "Test Mod"}
	body, _ := json.Marshal(mod)
	req := httptest.NewRequest("POST", "/mods", bytes.NewReader(body))

	// Generate token
	// This is a bit painful without AuthService, assume middleware works if we pass headers manually
	// Or we can skip middleware in this unit test if we test handler logic directly?
	// But RegisterRoutes applies middleware.
	// We should test handler function directly for logic, and integration test for middleware.
	// But `handleCreateMod` itself doesn't check auth, the router does.

	// If we test `handleCreateMod` directly:
	w := httptest.NewRecorder()
	h.handleCreateMod(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_CreateMod_Unauthorized(t *testing.T) {
	// This tests the RegisterRoutes wiring
	mockSvc := new(MockModService)
	h := NewHandler(mockSvc, new(MockGitHubService), "v1.0.0", nil)
	mw := middleware.NewAuthMiddleware("secret")
	h.SetAuthMiddleware(mw)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux, "")

	req := httptest.NewRequest("POST", "/mods", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateMod(t *testing.T) {
	mockSvc := new(MockModService)
	h := NewHandler(mockSvc, new(MockGitHubService), "v1.0.0", nil)

	modID := "test-mod"
	mod := modmgr.Mod{ID: modID, Name: "Updated Mod"}
	body, _ := json.Marshal(mod)
	req := httptest.NewRequest("PUT", "/mods/"+modID, bytes.NewReader(body))
	req.SetPathValue("modID", modID)

	mockSvc.On("UpdateMod", mock.Anything, mock.MatchedBy(func(m modmgr.Mod) bool {
		return m.ID == modID && m.Name == "Updated Mod"
	})).Return(nil)

	w := httptest.NewRecorder()
	h.handleUpdateMod(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_DeleteMod(t *testing.T) {
	mockSvc := new(MockModService)
	h := NewHandler(mockSvc, new(MockGitHubService), "v1.0.0", nil)

	modID := "test-mod"
	req := httptest.NewRequest("DELETE", "/mods/"+modID, nil)
	req.SetPathValue("modID", modID)

	mockSvc.On("DeleteMod", mock.Anything, modID).Return(nil)

	w := httptest.NewRecorder()
	h.handleDeleteMod(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_CreateMod_Error(t *testing.T) {
	mockSvc := new(MockModService)
	h := NewHandler(mockSvc, new(MockGitHubService), "v1.0.0", nil)

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/mods", bytes.NewReader([]byte("{invalid}")))
		w := httptest.NewRecorder()
		h.handleCreateMod(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("MissingFields", func(t *testing.T) {
		mod := modmgr.Mod{ID: ""}
		body, _ := json.Marshal(mod)
		req := httptest.NewRequest("POST", "/mods", bytes.NewReader(body))
		w := httptest.NewRecorder()
		h.handleCreateMod(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ServiceError", func(t *testing.T) {
		mod := modmgr.Mod{ID: "test", Name: "Test"}
		body, _ := json.Marshal(mod)
		req := httptest.NewRequest("POST", "/mods", bytes.NewReader(body))
		mockSvc.On("CreateMod", mock.Anything, mod).Return(errors.New("db error")).Once()
		w := httptest.NewRecorder()
		h.handleCreateMod(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandler_UpdateMod_Error(t *testing.T) {
	mockSvc := new(MockModService)
	h := NewHandler(mockSvc, new(MockGitHubService), "v1.0.0", nil)

	t.Run("IDMismatch", func(t *testing.T) {
		mod := modmgr.Mod{ID: "other"}
		body, _ := json.Marshal(mod)
		req := httptest.NewRequest("PUT", "/mods/test", bytes.NewReader(body))
		req.SetPathValue("modID", "test")
		w := httptest.NewRecorder()
		h.handleUpdateMod(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("MissingID", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/mods/", nil)
		req.SetPathValue("modID", "")
		w := httptest.NewRecorder()
		h.handleUpdateMod(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ServiceError", func(t *testing.T) {

		modID := "test"
		mod := modmgr.Mod{ID: modID, Name: "Test"}
		body, _ := json.Marshal(mod)
		req := httptest.NewRequest("PUT", "/mods/"+modID, bytes.NewReader(body))
		req.SetPathValue("modID", modID)
		mockSvc.On("UpdateMod", mock.Anything, mod).Return(errors.New("db error")).Once()
		w := httptest.NewRecorder()
		h.handleUpdateMod(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandler_DeleteMod_Error(t *testing.T) {
	mockSvc := new(MockModService)
	h := NewHandler(mockSvc, new(MockGitHubService), "v1.0.0", nil)

	t.Run("MissingID", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/mods/", nil)
		req.SetPathValue("modID", "")
		w := httptest.NewRecorder()
		h.handleDeleteMod(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ServiceError", func(t *testing.T) {

		modID := "test"
		req := httptest.NewRequest("DELETE", "/mods/"+modID, nil)
		req.SetPathValue("modID", modID)
		mockSvc.On("DeleteMod", mock.Anything, modID).Return(errors.New("db error")).Once()
		w := httptest.NewRecorder()
		h.handleDeleteMod(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandler_CreateModVersion(t *testing.T) {
	mockSvc := new(MockModService)
	h := NewHandler(mockSvc, new(MockGitHubService), "v1.0.0", nil)

	modID := "test-mod"
	version := modmgr.ModVersion{ID: "v1.0.0"}
	body, _ := json.Marshal(version)

	t.Run("Success_DefaultCompatibility", func(t *testing.T) {
		versionWithEmptyCompatible := modmgr.ModVersion{
			ID:    "v1.0.0",
			Files: []modmgr.ModFile{
				{
					URL:      "http://example.com/mod.zip",
					FileType: modmgr.FileTypeZip,
					// Compatible is empty, should be defaulted by handler
				},
			},
		}
		bodyWithEmptyCompatible, _ := json.Marshal(versionWithEmptyCompatible)

		expectedVersion := versionWithEmptyCompatible
		expectedVersion.ModID = modID
		expectedVersion.Files[0].Compatible = []aumgr.BinaryType{aumgr.BinaryType32Bit, aumgr.BinaryType64Bit}

		var capturedVersion modmgr.ModVersion
		mockSvc.On("CreateModVersion", mock.Anything, modID, mock.AnythingOfType("modmgr.ModVersion")).Run(func(args mock.Arguments) {
			capturedVersion = args.Get(2).(modmgr.ModVersion)
		}).Return(nil).Once()

		req := httptest.NewRequest("POST", "/mods/"+modID+"/versions", bytes.NewReader(bodyWithEmptyCompatible))
		req.SetPathValue("modID", modID)
		w := httptest.NewRecorder()
		h.handleCreateModVersion(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Assert captured version
		capturedVersion.CreatedAt = time.Time{} // Ignore CreatedAt for comparison
		expectedVersion.CreatedAt = time.Time{}

		assert.Equal(t, expectedVersion, capturedVersion)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/mods/"+modID+"/versions", bytes.NewReader(body))
		req.SetPathValue("modID", modID)
		mockSvc.On("CreateModVersion", mock.Anything, modID, mock.Anything).Return(nil).Once()
		w := httptest.NewRecorder()
		h.handleCreateModVersion(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("MissingModID", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/mods//versions", bytes.NewReader(body))
		req.SetPathValue("modID", "")
		w := httptest.NewRecorder()
		h.handleCreateModVersion(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/mods/"+modID+"/versions", bytes.NewReader([]byte("{invalid}")))
		req.SetPathValue("modID", modID)
		w := httptest.NewRecorder()
		h.handleCreateModVersion(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("MissingVersionID", func(t *testing.T) {
		version := modmgr.ModVersion{ID: ""}
		body, _ := json.Marshal(version)
		req := httptest.NewRequest("POST", "/mods/"+modID+"/versions", bytes.NewReader(body))
		req.SetPathValue("modID", modID)
		w := httptest.NewRecorder()
		h.handleCreateModVersion(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ServiceError", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/mods/"+modID+"/versions", bytes.NewReader(body))
		req.SetPathValue("modID", modID)
		mockSvc.On("CreateModVersion", mock.Anything, modID, mock.Anything).Return(errors.New("db error")).Once()
		w := httptest.NewRecorder()
		h.handleCreateModVersion(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandler_DeleteModVersion(t *testing.T) {
	mockSvc := new(MockModService)
	h := NewHandler(mockSvc, new(MockGitHubService), "v1.0.0", nil)

	modID := "test-mod"
	versionID := "v1.0.0"

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/mods/"+modID+"/versions/"+versionID, nil)
		req.SetPathValue("modID", modID)
		req.SetPathValue("versionID", versionID)
		mockSvc.On("DeleteModVersion", mock.Anything, modID, versionID).Return(nil).Once()
		w := httptest.NewRecorder()
		h.handleDeleteModVersion(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("MissingParams", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/mods///versions/", nil)
		req.SetPathValue("modID", "")
		req.SetPathValue("versionID", "")
		w := httptest.NewRecorder()
		h.handleDeleteModVersion(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ServiceError", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/mods/"+modID+"/versions/"+versionID, nil)
		req.SetPathValue("modID", modID)
		req.SetPathValue("versionID", versionID)
		mockSvc.On("DeleteModVersion", mock.Anything, modID, versionID).Return(errors.New("db error")).Once()
		w := httptest.NewRecorder()
		h.handleDeleteModVersion(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandler_CreateVersionFromGitHub(t *testing.T) {
	mockSvc := new(MockModService)
	h := NewHandler(mockSvc, new(MockGitHubService), "v1.0.0", nil)

	modID := "test-mod"
	releaseTag := "v1.0.0"
	githubRepo := "owner/repo"

	t.Run("Success_DefaultCompatibility", func(t *testing.T) {
		// Mock modService.GetMod
		mockSvc.On("GetMod", mock.Anything, modID).Return(&modmgr.Mod{ID: modID, GitHubRepo: githubRepo}, nil).Once()

		// Mock the githubService call
		mockGitHubService := new(MockGitHubService)
		h.githubService = mockGitHubService // Inject mock
		mockGitHubService.On("GetRelease", githubRepo[:strings.Index(githubRepo, "/")], githubRepo[strings.Index(githubRepo, "/")+1:], releaseTag).Return(&service.GitHubRelease{
			TagName: releaseTag,
			Assets: []service.GitHubAsset{
				{BrowserDownloadURL: "http://example.com/mod.zip", Name: "mod.zip"},
			},
		}, nil).Once()

		// Expect CreateModVersion to be called with correct compatibility
		expectedVersion := modmgr.ModVersion{
			ID:    releaseTag,
			ModID: modID,
			Files: []modmgr.ModFile{
				{
					URL:        "http://example.com/mod.zip",
					FileType:   modmgr.FileTypeZip,
					Compatible: []aumgr.BinaryType{aumgr.BinaryType32Bit, aumgr.BinaryType64Bit}, // Expected default compatibility
				},
			},
		}
		mockSvc.On("CreateModVersion", mock.Anything, modID, mock.MatchedBy(func(v modmgr.ModVersion) bool {
			// Deep compare everything except CreatedAt
			v.CreatedAt = time.Time{}
			expectedVersion.CreatedAt = time.Time{} // Clear for comparison
			return assert.Equal(t, expectedVersion, v)
		})).Return(nil).Once()

		reqBody := map[string]string{"tag": releaseTag}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/mods/"+modID+"/versions/from-github", bytes.NewReader(body))
		req.SetPathValue("modID", modID)
		w := httptest.NewRecorder()
		h.handleCreateVersionFromGitHub(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockSvc.AssertExpectations(t)
		mockGitHubService.AssertExpectations(t)
	})
}
