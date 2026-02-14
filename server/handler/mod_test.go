package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/ikafly144/au_mod_installer/server/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func TestHandler_CreateMod(t *testing.T) {
	mockSvc := new(MockModService)
	h := NewHandler(mockSvc, "v1.0.0", nil)

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
	h := NewHandler(mockSvc, "v1.0.0", nil)
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
	h := NewHandler(mockSvc, "v1.0.0", nil)

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
	h := NewHandler(mockSvc, "v1.0.0", nil)

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
	h := NewHandler(mockSvc, "v1.0.0", nil)

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
	h := NewHandler(mockSvc, "v1.0.0", nil)

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
	h := NewHandler(mockSvc, "v1.0.0", nil)

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
	h := NewHandler(mockSvc, "v1.0.0", nil)

	modID := "test-mod"
	version := modmgr.ModVersion{ID: "v1.0.0"}
	body, _ := json.Marshal(version)

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
	h := NewHandler(mockSvc, "v1.0.0", nil)

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
