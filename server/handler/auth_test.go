package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ikafly144/au_mod_installer/server/model"
	"github.com/ikafly144/au_mod_installer/server/repository"
	"github.com/ikafly144/au_mod_installer/server/service"
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

func (m *MockUserRepo) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	args := m.Called(ctx, username)
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

func TestAuthHandler_Register(t *testing.T) {
	mockRepo := new(MockUserRepo)
	svc := service.NewAuthService(mockRepo, "secret")
	h := NewAuthHandler(svc)

	// Mock GetUserByUsername to return error (not found) for first check, then return user for second check
	// This is tricky with testify/mock if arguments are same.
	// But Register calls GetUserByUsername twice. First to check if exists, second to return it.
	// 1. check: returns nil, nil (user not found)
	// 2. return: returns user (good)

	// Since arguments are same "newuser", we need to sequence them?
	// testify/mock .Once() helps.

	mockRepo.On("GetUserByUsername", mock.Anything, "newuser").Return(nil, nil).Once()
	mockRepo.On("CreateUser", mock.Anything, mock.MatchedBy(func(u model.User) bool {
		return u.Username == "newuser"
	})).Return(nil)
	mockRepo.On("GetUserByUsername", mock.Anything, "newuser").Return(&model.User{ID: 1, Username: "newuser"}, nil).Once()

	reqBody := service.RegisterRequest{
		Username:    "newuser",
		Password:    "password",
		DisplayName: "New User",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.handleRegister(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var user model.User
	err := json.NewDecoder(w.Body).Decode(&user)
	assert.NoError(t, err)
	assert.Equal(t, "newuser", user.Username)

	mockRepo.AssertExpectations(t)
}
