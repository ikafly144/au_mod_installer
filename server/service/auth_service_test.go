package service

import (
	"context"
	"testing"

	"github.com/ikafly144/au_mod_installer/server/model"
	"github.com/ikafly144/au_mod_installer/server/repository"
	"github.com/stretchr/testify/mock"
	"github.com/tj/assert"
	"golang.org/x/crypto/bcrypt"
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

func TestAuthService_Login(t *testing.T) {
	mockRepo := new(MockUserRepo)
	service := NewAuthService(mockRepo, "secret")
	ctx := context.Background()

	password := "password"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)

	user := &model.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: string(hash),
	}

	mockRepo.On("GetUserByUsername", ctx, "testuser").Return(user, nil)

	resp, err := service.Login(ctx, LoginRequest{Username: "testuser", Password: password})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, user, resp.User)

	mockRepo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	mockRepo := new(MockUserRepo)
	service := NewAuthService(mockRepo, "secret")
	ctx := context.Background()

	password := "password"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)

	user := &model.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: string(hash),
	}

	mockRepo.On("GetUserByUsername", ctx, "testuser").Return(user, nil)

	resp, err := service.Login(ctx, LoginRequest{Username: "testuser", Password: "wrongpassword"})
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidCredentials, err)
	assert.Nil(t, resp)

	mockRepo.AssertExpectations(t)
}
