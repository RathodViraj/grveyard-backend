package buy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"grveyard/pkg/response"
)

type mockBuyService struct {
	mock.Mock
}

func (m *mockBuyService) MarkAssetSold(ctx context.Context, assetID int64) error {
	args := m.Called(ctx, assetID)
	return args.Error(0)
}

func (m *mockBuyService) UnlistAsset(ctx context.Context, assetID int64) error {
	args := m.Called(ctx, assetID)
	return args.Error(0)
}

func (m *mockBuyService) MarkStartupSold(ctx context.Context, startupID int64) error {
	args := m.Called(ctx, startupID)
	return args.Error(0)
}

func (m *mockBuyService) UnlistStartup(ctx context.Context, startupID int64) error {
	args := m.Called(ctx, startupID)
	return args.Error(0)
}

func setupBuyRouter(service BuyService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewBuyHandler(service)
	h.RegisterRoutes(r)
	return r
}

func TestBuyHandler_MarkAssetSold_Success(t *testing.T) {
	svc := new(mockBuyService)
	r := setupBuyRouter(svc)

	svc.On("MarkAssetSold", mock.Anything, int64(1)).Return(nil)

	req := httptest.NewRequest(http.MethodPatch, "/assets/1/mark-sold", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "asset marked as sold", resp.Message)

	svc.AssertExpectations(t)
}

func TestBuyHandler_MarkAssetSold_AlreadySold(t *testing.T) {
	svc := new(mockBuyService)
	r := setupBuyRouter(svc)

	svc.On("MarkAssetSold", mock.Anything, int64(1)).Return(ErrAlreadySold)

	req := httptest.NewRequest(http.MethodPatch, "/assets/1/mark-sold", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "asset already marked as sold", resp.Message)

	svc.AssertExpectations(t)
}

func TestBuyHandler_UnlistAsset_NotFound(t *testing.T) {
	svc := new(mockBuyService)
	r := setupBuyRouter(svc)

	svc.On("UnlistAsset", mock.Anything, int64(2)).Return(ErrNotFound)

	req := httptest.NewRequest(http.MethodPatch, "/assets/2/unlist", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "asset not found", resp.Message)

	svc.AssertExpectations(t)
}

func TestBuyHandler_MarkStartupSold_NotFound(t *testing.T) {
	svc := new(mockBuyService)
	r := setupBuyRouter(svc)

	svc.On("MarkStartupSold", mock.Anything, int64(3)).Return(ErrNotFound)

	req := httptest.NewRequest(http.MethodPatch, "/startups/3/mark-sold", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "startup not found", resp.Message)

	svc.AssertExpectations(t)
}

func TestBuyHandler_UnlistStartup_InvalidID(t *testing.T) {
	svc := new(mockBuyService)
	r := setupBuyRouter(svc)

	req := httptest.NewRequest(http.MethodPatch, "/startups/abc/unlist", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp response.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "invalid startup id", resp.Message)

	svc.AssertNotCalled(t, "UnlistStartup", mock.Anything, mock.Anything)
}
