package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"grveyard/pkg/response"
)

type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) CreateUser(ctx context.Context, name, email, role, password, profilePicURL, uuid string) (User, error) {
	args := m.Called(ctx, name, email, role, password, profilePicURL, uuid)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserService) UpdateUser(ctx context.Context, u User) (User, error) {
	args := m.Called(ctx, u)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserService) UpdateUserByUUID(ctx context.Context, currentUUID string, u User) (User, error) {
	args := m.Called(ctx, currentUUID, u)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserService) DeleteUser(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockUserService) DeleteUserByUUID(ctx context.Context, uuid string) error {
	args := m.Called(ctx, uuid)
	return args.Error(0)
}

func (m *mockUserService) GetUserByID(ctx context.Context, id int64) (User, error) {
	args := m.Called(ctx, id)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserService) GetUserByUUID(ctx context.Context, uuid string) (User, error) {
	args := m.Called(ctx, uuid)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserService) GetUserByEmail(ctx context.Context, email string) (User, error) {
	args := m.Called(ctx, email)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserService) ListUsers(ctx context.Context, page, limit int) ([]User, int64, error) {
	args := m.Called(ctx, page, limit)
	users, _ := args.Get(0).([]User)
	return users, args.Get(1).(int64), args.Error(2)
}

func (m *mockUserService) Login(ctx context.Context, email, password string) (User, error) {
	args := m.Called(ctx, email, password)
	user, _ := args.Get(0).(User)
	return user, args.Error(1)
}

func (m *mockUserService) VerifyEmail(ctx context.Context, email string) (User, bool, error) {
	args := m.Called(ctx, email)
	user, _ := args.Get(0).(User)
	return user, args.Bool(1), args.Error(2)
}

func setupUserRouter(service UserService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewUserHandler(service)
	h.RegisterRoutes(r)
	return r
}

func TestUserHandler_CreateUser_Success(t *testing.T) {
	svc := new(mockUserService)
	r := setupUserRouter(svc)

	expected := User{ID: 1, Name: "Alice", Email: "a@example.com", Role: "buyer"}
	svc.On("CreateUser", mock.Anything, "Alice", "a@example.com", "buyer", "pass", "pic", "uuid").Return(expected, nil)

	reqBody := `{"name":"Alice","email":"a@example.com","role":"buyer","password":"pass","profile_pic_url":"pic","uuid":"uuid"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "user created", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 1, data["id"])
	require.Equal(t, "Alice", data["name"])

	svc.AssertExpectations(t)
}

func TestUserHandler_CreateUser_InvalidPayload(t *testing.T) {
	svc := new(mockUserService)
	r := setupUserRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"email":"a@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "invalid request payload", resp.Message)

	svc.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUserHandler_UpdateUser_NotFound(t *testing.T) {
	svc := new(mockUserService)
	r := setupUserRouter(svc)

	svc.On("UpdateUserByUUID", mock.Anything, "uuid-1", mock.Anything).Return(User{}, ErrUserNotFound)

	req := httptest.NewRequest(http.MethodPut, "/users/uuid-1", strings.NewReader(`{"name":"New"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "user not found", resp.Message)

	svc.AssertExpectations(t)
}

func TestUserHandler_DeleteUser_NotFound(t *testing.T) {
	svc := new(mockUserService)
	r := setupUserRouter(svc)

	svc.On("DeleteUserByUUID", mock.Anything, "uuid-x").Return(ErrUserNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/users/uuid-x", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "user not found", resp.Message)

	svc.AssertExpectations(t)
}

func TestUserHandler_Login_InvalidCredentials(t *testing.T) {
	svc := new(mockUserService)
	r := setupUserRouter(svc)

	svc.On("Login", mock.Anything, "a@example.com", "bad").Return(User{}, errors.New("invalid credentials"))

	req := httptest.NewRequest(http.MethodPost, "/users/login", strings.NewReader(`{"email":"a@example.com","password":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "invalid credentials", resp.Message)

	svc.AssertExpectations(t)
}

func TestUserHandler_GetUserByUUID_Success(t *testing.T) {
	svc := new(mockUserService)
	r := setupUserRouter(svc)

	svc.On("GetUserByUUID", mock.Anything, "uuid-1").Return(User{ID: 1, UUID: "uuid-1", Name: "A"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/uuid-1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "user fetched", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "uuid-1", data["uuid"])

	svc.AssertExpectations(t)
}

func TestUserHandler_ListUsers_Success(t *testing.T) {
	svc := new(mockUserService)
	r := setupUserRouter(svc)

	items := []User{{ID: 1, Name: "A"}}
	svc.On("ListUsers", mock.Anything, 2, 1).Return(items, int64(1), nil)

	req := httptest.NewRequest(http.MethodGet, "/users?page=2&limit=1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "users listed", resp.Message)
	require.WithinDuration(t, time.Now(), resp.CreatedAt, time.Minute)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 1, data["total"])
	require.EqualValues(t, 2, data["page"])
	require.EqualValues(t, 1, data["limit"])

	itemsRaw, ok := data["items"].([]any)
	require.True(t, ok)
	require.Len(t, itemsRaw, 1)

	item, ok := itemsRaw[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "A", item["name"])

	svc.AssertExpectations(t)
}

func TestUserHandler_VerifyUser_SuccessWithinWindow(t *testing.T) {
	svc := new(mockUserService)
	r := setupUserRouter(svc)

	svc.On("VerifyEmail", mock.Anything, "a@example.com").Return(User{Email: "a@example.com"}, true, nil)

	req := httptest.NewRequest(http.MethodPost, "/users/verify", strings.NewReader(`{"email":"a@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "user verified", resp.Message)
	svc.AssertExpectations(t)
}

func TestUserHandler_VerifyUser_NotVerified(t *testing.T) {
	svc := new(mockUserService)
	r := setupUserRouter(svc)

	svc.On("VerifyEmail", mock.Anything, "a@example.com").Return(User{}, false, nil)

	req := httptest.NewRequest(http.MethodPost, "/users/verify", strings.NewReader(`{"email":"a@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "user not verified", resp.Message)
	svc.AssertExpectations(t)
}

func TestUserHandler_CheckVerification_Boolean(t *testing.T) {
	svc := new(mockUserService)
	r := setupUserRouter(svc)

	now := time.Now()
	svc.On("GetUserByEmail", mock.Anything, "a@example.com").Return(User{VerifiedAt: &now}, nil)

	req := httptest.NewRequest(http.MethodPost, "/users/checkVerification", strings.NewReader(`{"email":"a@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "true", strings.TrimSpace(w.Body.String()))

	svc.AssertNotCalled(t, "VerifyEmail", mock.Anything, mock.Anything)
	svc.AssertExpectations(t)
}

func TestUserHandler_CheckVerification_Unverified(t *testing.T) {
	svc := new(mockUserService)
	r := setupUserRouter(svc)

	svc.On("GetUserByEmail", mock.Anything, "a@example.com").Return(User{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/users/checkVerification", strings.NewReader(`{"email":"a@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "false", strings.TrimSpace(w.Body.String()))

	svc.AssertNotCalled(t, "VerifyEmail", mock.Anything, mock.Anything)
	svc.AssertExpectations(t)
}
