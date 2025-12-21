package assets

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

type mockAssetService struct {
	mock.Mock
}

func (m *mockAssetService) CreateAsset(ctx context.Context, input Asset) (Asset, error) {
	args := m.Called(ctx, input)
	asset, _ := args.Get(0).(Asset)
	return asset, args.Error(1)
}

func (m *mockAssetService) UpdateAsset(ctx context.Context, input Asset) (Asset, error) {
	args := m.Called(ctx, input)
	asset, _ := args.Get(0).(Asset)
	return asset, args.Error(1)
}

func (m *mockAssetService) DeleteAsset(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockAssetService) GetAssetByID(ctx context.Context, id int64) (Asset, error) {
	args := m.Called(ctx, id)
	asset, _ := args.Get(0).(Asset)
	return asset, args.Error(1)
}

func (m *mockAssetService) ListAssets(ctx context.Context, filters AssetFilters, page, limit int) ([]Asset, int64, error) {
	args := m.Called(ctx, filters, page, limit)
	assets, _ := args.Get(0).([]Asset)
	return assets, args.Get(1).(int64), args.Error(2)
}

func (m *mockAssetService) ListAssetsByUser(ctx context.Context, userUUID string, page, limit int) ([]Asset, int64, error) {
	args := m.Called(ctx, userUUID, page, limit)
	assets, _ := args.Get(0).([]Asset)
	return assets, args.Get(1).(int64), args.Error(2)
}

func (m *mockAssetService) DeleteAllAssets(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockAssetService) DeleteAllAssetsByUserUUID(ctx context.Context, userUUID string) error {
	args := m.Called(ctx, userUUID)
	return args.Error(0)
}

func setupAssetRouter(service AssetService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAssetHandler(service)
	h.RegisterRoutes(r)
	return r
}

func TestAssetHandler_CreateAsset_Success(t *testing.T) {
	svc := new(mockAssetService)
	r := setupAssetRouter(svc)

	expected := Asset{ID: 1, UserUUID: "uuid-1", Title: "Asset", AssetType: "research", IsNegotiable: true, IsActive: true}
	svc.On("CreateAsset", mock.Anything, mock.MatchedBy(func(a Asset) bool {
		return a.UserUUID == "uuid-1" && a.Title == "Asset" && a.AssetType == "research"
	})).Return(expected, nil)

	reqBody := `{"user_uuid":"uuid-1","title":"Asset","description":"d","asset_type":"research","image_url":"img","price":10,"is_negotiable":true,"is_sold":false}`
	req := httptest.NewRequest(http.MethodPost, "/assets", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "asset created", resp.Message)

	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 1, data["id"])
	require.Equal(t, "Asset", data["title"])

	svc.AssertExpectations(t)
}

func TestAssetHandler_CreateAsset_InvalidType(t *testing.T) {
	svc := new(mockAssetService)
	r := setupAssetRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/assets", strings.NewReader(`{"user_uuid":"uuid-1","title":"Asset","asset_type":"weird"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "invalid asset_type", resp.Message)

	svc.AssertNotCalled(t, "CreateAsset", mock.Anything, mock.Anything)
}

func TestAssetHandler_CreateAsset_NegativePrice(t *testing.T) {
	svc := new(mockAssetService)
	r := setupAssetRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/assets", strings.NewReader(`{"user_uuid":"uuid-1","title":"Asset","asset_type":"research","price":-1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "price cannot be negative", resp.Message)

	svc.AssertNotCalled(t, "CreateAsset", mock.Anything, mock.Anything)
}

func TestAssetHandler_UpdateAsset_NotFound(t *testing.T) {
	svc := new(mockAssetService)
	r := setupAssetRouter(svc)

	svc.On("UpdateAsset", mock.Anything, mock.Anything).Return(Asset{}, ErrAssetNotFound)

	req := httptest.NewRequest(http.MethodPut, "/assets/1", strings.NewReader(`{"title":"Asset","asset_type":"research"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "asset not found", resp.Message)

	svc.AssertExpectations(t)
}

// func TestAssetHandler_ListAssets_Success(t *testing.T) {
// 	svc := new(mockAssetService)
// 	r := setupAssetRouter(svc)

// 	items := []Asset{{ID: 1, Title: "A"}}
// 	svc.On("ListAssets", mock.Anything, mock.Anything, 2, 1).Return(items, int64(1), nil)

// 	req := httptest.NewRequest(http.MethodGet, "/assets?page=2&limit=1&asset_type=research&is_sold=false", nil)
// 	w := httptest.NewRecorder()

// 	r.ServeHTTP(w, req)

// 	require.Equal(t, http.StatusOK, w.Code)
// 	var resp response.APIResponse
// 	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
// 	require.True(t, resp.Success)
// 	require.Equal(t, "assets listed", resp.Message)
// 	require.WithinDuration(t, time.Now(), resp.CreatedAt, time.Minute)

// 	data, ok := resp.Data.(map[string]any)
// 	require.True(t, ok)
// 	require.EqualValues(t, 1, data["total"])

// 	itemsRaw, ok := data["items"].([]any)
// 	require.True(t, ok)
// 	require.Len(t, itemsRaw, 1)
// }

func TestAssetHandler_ListAssetsByUser_InvalidUUID(t *testing.T) {
	svc := new(mockAssetService)
	r := setupAssetRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/users//assets", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "invalid user uuid", resp.Message)

	svc.AssertNotCalled(t, "ListAssetsByUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
