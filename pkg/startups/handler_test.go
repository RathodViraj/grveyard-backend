package startups

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"grveyard/pkg/response"
)

type mockStartupService struct {
	mock.Mock
}

func (m *mockStartupService) CreateStartup(ctx context.Context, input Startup) (Startup, error) {
	args := m.Called(ctx, input)
	startup, _ := args.Get(0).(Startup)
	return startup, args.Error(1)
}

func (m *mockStartupService) UpdateStartup(ctx context.Context, input Startup) (Startup, error) {
	args := m.Called(ctx, input)
	startup, _ := args.Get(0).(Startup)
	return startup, args.Error(1)
}

func (m *mockStartupService) DeleteStartup(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockStartupService) GetStartupByID(ctx context.Context, id int64) (Startup, error) {
	args := m.Called(ctx, id)
	startup, _ := args.Get(0).(Startup)
	return startup, args.Error(1)
}

func (m *mockStartupService) ListStartups(ctx context.Context, page, limit int) ([]Startup, int64, error) {
	args := m.Called(ctx, page, limit)
	startups, _ := args.Get(0).([]Startup)
	return startups, args.Get(1).(int64), args.Error(2)
}

func (m *mockStartupService) DeleteAllStartups(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func setupRouter(service StartupService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewStartupHandler(service)
	h.RegisterRoutes(r)
	return r
}

func TestStartupHandler_CreateStartup_Success(t *testing.T) {
	svc := new(mockStartupService)
	r := setupRouter(svc)

	expected := Startup{ID: 1, Name: "Acme", OwnerUUID: "user-uuid-1", Status: "active"}
	svc.On("CreateStartup", mock.Anything, mock.MatchedBy(func(input Startup) bool {
		return input.Name == "Acme" && input.OwnerUUID == "user-uuid-1" && input.Status == "active"
	})).Return(expected, nil)

	reqBody := `{"name":"Acme","description":"desc","logo_url":"logo","owner_uuid":"user-uuid-1","status":"active"}`
	req := httptest.NewRequest(http.MethodPost, "/startups", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "startup created", resp.Message)
	require.False(t, resp.CreatedAt.IsZero())

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 1, data["id"])
	require.Equal(t, "Acme", data["name"])

	svc.AssertExpectations(t)
}

func TestStartupHandler_CreateStartup_InvalidPayload(t *testing.T) {
	svc := new(mockStartupService)
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/startups", strings.NewReader(`{"owner_uuid":"user-uuid-1"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "invalid request payload", resp.Message)

	svc.AssertNotCalled(t, "CreateStartup", mock.Anything, mock.Anything)
}

func TestStartupHandler_CreateStartup_InvalidStatus(t *testing.T) {
	svc := new(mockStartupService)
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/startups", strings.NewReader(`{"name":"Acme","owner_uuid":"user-uuid-1","status":"weird"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "invalid status", resp.Message)

	svc.AssertNotCalled(t, "CreateStartup", mock.Anything, mock.Anything)
}

func TestStartupHandler_UpdateStartup_InvalidID(t *testing.T) {
	svc := new(mockStartupService)
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/startups/abc", strings.NewReader(`{"name":"Acme"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "invalid startup id", resp.Message)

	svc.AssertNotCalled(t, "UpdateStartup", mock.Anything, mock.Anything)
}

func TestStartupHandler_DeleteStartup_NotFound(t *testing.T) {
	svc := new(mockStartupService)
	r := setupRouter(svc)

	svc.On("DeleteStartup", mock.Anything, int64(42)).Return(ErrStartupNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/startups/42", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "startup not found", resp.Message)

	svc.AssertExpectations(t)
}

// func TestStartupHandler_ListStartups_Success(t *testing.T) {
// 	svc := new(mockStartupService)
// 	r := setupRouter(svc)

// 	expectedItems := []Startup{{ID: 1, Name: "Acme", OwnerID: 1, Status: "active"}}
// 	svc.On("ListStartups", mock.Anything, 2, 1).Return(expectedItems, int64(1), nil)

// 	req := httptest.NewRequest(http.MethodGet, "/startups?page=2&limit=1", nil)
// 	w := httptest.NewRecorder()

// 	r.ServeHTTP(w, req)

// 	require.Equal(t, http.StatusOK, w.Code)
// 	var resp response.APIResponse
// 	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
// 	require.True(t, resp.Success)
// 	require.Equal(t, "startups listed", resp.Message)
// 	require.WithinDuration(t, time.Now(), resp.CreatedAt, time.Minute)

// 	data, ok := resp.Data.(map[string]any)
// 	require.True(t, ok)
// 	require.EqualValues(t, 1, data["total"])
// 	require.EqualValues(t, 2, data["page"])
// 	require.EqualValues(t, 1, data["limit"])

// 	items, ok := data["items"].([]any)
// 	require.True(t, ok)
// 	require.Len(t, items, 1)

// 	item, ok := items[0].(map[string]any)
// 	require.True(t, ok)
// 	require.EqualValues(t, 1, item["id"])
// 	require.Equal(t, "Acme", item["name"])

// 	svc.AssertExpectations(t)
// }
