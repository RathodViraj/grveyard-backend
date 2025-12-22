package users

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgconn"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepository struct {
	mock.Mock
}

func (m *mockUserRepository) CreateUser(ctx context.Context, name, email, role, passwordHash, profilePicURL, uuid string) (User, error) {
	args := m.Called(ctx, name, email, role, passwordHash, profilePicURL, uuid)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserRepository) UpdateUser(ctx context.Context, u User) (User, error) {
	args := m.Called(ctx, u)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserRepository) UpdateUserByUUID(ctx context.Context, currentUUID string, u User) (User, error) {
	args := m.Called(ctx, currentUUID, u)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserRepository) DeleteUser(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockUserRepository) DeleteUserByUUID(ctx context.Context, uuid string) error {
	args := m.Called(ctx, uuid)
	return args.Error(0)
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, id int64) (User, error) {
	args := m.Called(ctx, id)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserRepository) GetUserByUUID(ctx context.Context, uuid string) (User, error) {
	args := m.Called(ctx, uuid)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	args := m.Called(ctx, email)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserRepository) GetUserByEmailIncludingDeleted(ctx context.Context, email string) (User, error) {
	args := m.Called(ctx, email)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserRepository) ReviveUserByEmail(ctx context.Context, email, name, role, passwordHash, profilePicURL, uuid string) (User, error) {
	args := m.Called(ctx, email, name, role, passwordHash, profilePicURL, uuid)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserRepository) ListUsers(ctx context.Context, limit, offset int) ([]User, int64, error) {
	args := m.Called(ctx, limit, offset)
	users, _ := args.Get(0).([]User)
	return users, args.Get(1).(int64), args.Error(2)
}

func (m *mockUserRepository) GetUserAuthByEmail(ctx context.Context, email string) (int64, string, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(int64), args.String(1), args.Error(2)
}

func (m *mockUserRepository) UpdateVerifiedAtByEmail(ctx context.Context, email string, ts time.Time) error {
	args := m.Called(ctx, email, ts)
	return args.Error(0)
}

func TestUserService_CreateUser_InvalidRole(t *testing.T) {
	repo := new(mockUserRepository)
	service := NewUserService(repo)

	_, err := service.CreateUser(context.Background(), "Name", "a@example.com", "wrong", "pass", "", "uuid")

	require.EqualError(t, err, "invalid role")
	repo.AssertExpectations(t)
}

func TestUserService_CreateUser_DuplicateEmail(t *testing.T) {
	repo := new(mockUserRepository)
	service := NewUserService(repo)

	repo.On("CreateUser", mock.Anything, "Name", "a@example.com", "buyer", mock.Anything, "", "uuid").Return(User{}, &pgconn.PgError{Code: "23505"})

	_, err := service.CreateUser(context.Background(), "Name", "a@example.com", "buyer", "pass", "", "uuid")

	require.EqualError(t, err, "user exists with that email")
	repo.AssertExpectations(t)
}

func TestUserService_UpdateUser_InvalidRole(t *testing.T) {
	repo := new(mockUserRepository)
	service := NewUserService(repo)

	_, err := service.UpdateUser(context.Background(), User{ID: 1, Name: "Bob", Role: "invalid"})

	require.EqualError(t, err, "invalid role")
	repo.AssertExpectations(t)
}

func TestUserService_UpdateUserByUUID_FillUUID(t *testing.T) {
	repo := new(mockUserRepository)
	service := NewUserService(repo)

	repo.On("UpdateUserByUUID", mock.Anything, "current", mock.MatchedBy(func(u User) bool {
		return u.UUID == "current" && u.Name == "Bob"
	})).Return(User{ID: 1, Name: "Bob", UUID: "current"}, nil)

	_, err := service.UpdateUserByUUID(context.Background(), "current", User{Name: "Bob"})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUserService_ListUsers_Defaults(t *testing.T) {
	repo := new(mockUserRepository)
	service := NewUserService(repo)

	repo.On("ListUsers", mock.Anything, 10, 0).Return([]User{}, int64(0), nil)

	_, _, err := service.ListUsers(context.Background(), 0, 0)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUserService_Login_InvalidPassword(t *testing.T) {
	repo := new(mockUserRepository)
	service := NewUserService(repo)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	require.NoError(t, err)

	repo.On("GetUserAuthByEmail", mock.Anything, "a@example.com").Return(int64(1), string(hash), nil)

	_, err = service.Login(context.Background(), "a@example.com", "wrong")

	require.EqualError(t, err, "invalid credentials")
	repo.AssertNotCalled(t, "GetUserByID", mock.Anything, mock.Anything)
}

func TestUserService_Login_UserNotFound(t *testing.T) {
	repo := new(mockUserRepository)
	service := NewUserService(repo)

	repo.On("GetUserAuthByEmail", mock.Anything, "a@example.com").Return(int64(0), "", ErrUserNotFound)

	_, err := service.Login(context.Background(), "a@example.com", "secret")

	require.EqualError(t, err, "invalid credentials")
	repo.AssertExpectations(t)
}

func TestUserService_Login_Success(t *testing.T) {
	repo := new(mockUserRepository)
	service := NewUserService(repo)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	require.NoError(t, err)

	repo.On("GetUserAuthByEmail", mock.Anything, "a@example.com").Return(int64(10), string(hash), nil)
	repo.On("GetUserByID", mock.Anything, int64(10)).Return(User{ID: 10, Email: "a@example.com"}, nil)

	u, err := service.Login(context.Background(), "a@example.com", "secret")

	require.NoError(t, err)
	require.Equal(t, int64(10), u.ID)
	repo.AssertExpectations(t)
}

func TestUserService_CheckAndUpdateVerification_OutsideWindow(t *testing.T) {
	repo := new(mockUserRepository)
	service := NewUserService(repo)

	now := time.Now().Add(-40 * 24 * time.Hour)
	user := User{Email: "a@example.com", VerifiedAt: &now}

	repo.On("GetUserByEmail", mock.Anything, "a@example.com").Return(user, nil)

	verified, err := service.CheckAndUpdateVerification(context.Background(), "a@example.com")

	require.NoError(t, err)
	require.False(t, verified)
	repo.AssertNotCalled(t, "UpdateVerifiedAtByEmail", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestUserService_CheckAndUpdateVerification_WithinWindow(t *testing.T) {
	repo := new(mockUserRepository)
	service := NewUserService(repo)

	verified := time.Now().Add(-10 * 24 * time.Hour)
	user := User{Email: "a@example.com", VerifiedAt: &verified}

	repo.On("GetUserByEmail", mock.Anything, "a@example.com").Return(user, nil)
	repo.On("UpdateVerifiedAtByEmail", mock.Anything, "a@example.com", mock.Anything).Return(nil)

	within, err := service.CheckAndUpdateVerification(context.Background(), "a@example.com")

	require.NoError(t, err)
	require.True(t, within)
	repo.AssertExpectations(t)
}
