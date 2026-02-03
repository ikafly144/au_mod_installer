package service

import (
	"context"
	"testing"

	"github.com/ikafly144/au_mod_installer/pkg/modmgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockModRepository struct {
	mock.Mock
}

func (m *MockModRepository) GetMod(ctx context.Context, modID string) (*modmgr.Mod, error) {
	args := m.Called(ctx, modID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*modmgr.Mod), args.Error(1)
}

func (m *MockModRepository) GetModList(ctx context.Context, limit int, after string, before string) ([]modmgr.Mod, error) {
	args := m.Called(ctx, limit, after, before)
	return args.Get(0).([]modmgr.Mod), args.Error(1)
}

func (m *MockModRepository) GetModVersion(ctx context.Context, modID string, versionID string) (*modmgr.ModVersion, error) {
	args := m.Called(ctx, modID, versionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*modmgr.ModVersion), args.Error(1)
}

func (m *MockModRepository) GetModVersions(ctx context.Context, modID string, limit int, after string) ([]modmgr.ModVersion, error) {
	args := m.Called(ctx, modID, limit, after)
	return args.Get(0).([]modmgr.ModVersion), args.Error(1)
}

func (m *MockModRepository) CreateMod(ctx context.Context, mod modmgr.Mod) error {
	args := m.Called(ctx, mod)
	return args.Error(0)
}

func (m *MockModRepository) UpdateMod(ctx context.Context, mod modmgr.Mod) error {
	args := m.Called(ctx, mod)
	return args.Error(0)
}

func (m *MockModRepository) SetMod(ctx context.Context, mod modmgr.Mod) error {
	args := m.Called(ctx, mod)
	return args.Error(0)
}

func (m *MockModRepository) CreateModVersion(ctx context.Context, modID string, version modmgr.ModVersion) error {
	args := m.Called(ctx, modID, version)
	return args.Error(0)
}

func (m *MockModRepository) SetModVersion(ctx context.Context, modID string, version modmgr.ModVersion) error {
	args := m.Called(ctx, modID, version)
	return args.Error(0)
}

func (m *MockModRepository) Close() {
	m.Called()
}

func (m *MockModRepository) GetAllMods(ctx context.Context) ([]modmgr.Mod, error) {
	args := m.Called(ctx)
	return args.Get(0).([]modmgr.Mod), args.Error(1)
}

func (m *MockModRepository) DeleteMod(ctx context.Context, modID string) error {
	args := m.Called(ctx, modID)
	return args.Error(0)
}

func (m *MockModRepository) GetAllModVersions(ctx context.Context, modID string) ([]modmgr.ModVersion, error) {
	args := m.Called(ctx, modID)
	return args.Get(0).([]modmgr.ModVersion), args.Error(1)
}

func (m *MockModRepository) DeleteModVersion(ctx context.Context, modID, versionID string) error {
	args := m.Called(ctx, modID, versionID)
	return args.Error(0)
}

func (m *MockModRepository) DeleteVersion(ctx context.Context, modID, versionID string) error {
	args := m.Called(ctx, modID, versionID)
	return args.Error(0)
}

func TestModService_CRUD(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockModRepository)
	svc := NewModServiceWithRepo(mockRepo)

	t.Run("CreateMod", func(t *testing.T) {
		mod := modmgr.Mod{ID: "test"}
		mockRepo.On("CreateMod", ctx, mod).Return(nil).Once()
		err := svc.CreateMod(ctx, mod)
		assert.NoError(t, err)
	})

	t.Run("UpdateMod", func(t *testing.T) {
		mod := modmgr.Mod{ID: "test"}
		mockRepo.On("UpdateMod", ctx, mod).Return(nil).Once()
		err := svc.UpdateMod(ctx, mod)
		assert.NoError(t, err)
	})

	t.Run("DeleteMod", func(t *testing.T) {
		modID := "test"
		mockRepo.On("DeleteMod", ctx, modID).Return(nil).Once()
		err := svc.DeleteMod(ctx, modID)
		assert.NoError(t, err)
	})

	t.Run("CreateModVersion", func(t *testing.T) {
		modID := "test"
		version := modmgr.ModVersion{ID: "v1"}
		mockRepo.On("CreateModVersion", ctx, modID, version).Return(nil).Once()
		err := svc.CreateModVersion(ctx, modID, version)
		assert.NoError(t, err)
	})

	t.Run("DeleteModVersion", func(t *testing.T) {
		modID := "test"
		versionID := "v1"
		mockRepo.On("DeleteModVersion", ctx, modID, versionID).Return(nil).Once()
		err := svc.DeleteModVersion(ctx, modID, versionID)
		assert.NoError(t, err)
	})

	mockRepo.AssertExpectations(t)
}
