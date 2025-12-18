package buy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockBuyRepository struct {
	mock.Mock
}

func (m *mockBuyRepository) MarkAssetSold(ctx context.Context, assetID int64) error {
	args := m.Called(ctx, assetID)
	return args.Error(0)
}

func (m *mockBuyRepository) UnlistAsset(ctx context.Context, assetID int64) error {
	args := m.Called(ctx, assetID)
	return args.Error(0)
}

func (m *mockBuyRepository) MarkStartupSold(ctx context.Context, startupID int64) error {
	args := m.Called(ctx, startupID)
	return args.Error(0)
}

func (m *mockBuyRepository) UnlistStartup(ctx context.Context, startupID int64) error {
	args := m.Called(ctx, startupID)
	return args.Error(0)
}

func (m *mockBuyRepository) GetAssetStatus(ctx context.Context, assetID int64) (bool, bool, error) {
	args := m.Called(ctx, assetID)
	return args.Bool(0), args.Bool(1), args.Error(2)
}

func (m *mockBuyRepository) GetStartupStatus(ctx context.Context, startupID int64) (string, error) {
	args := m.Called(ctx, startupID)
	return args.String(0), args.Error(1)
}

func TestBuyService_MarkAssetSold_AlreadySold(t *testing.T) {
	repo := new(mockBuyRepository)
	service := NewBuyService(repo)

	repo.On("GetAssetStatus", mock.Anything, int64(1)).Return(true, true, nil)

	err := service.MarkAssetSold(context.Background(), 1)

	require.ErrorIs(t, err, ErrAlreadySold)
	repo.AssertExpectations(t)
}

func TestBuyService_MarkAssetSold_Inactive(t *testing.T) {
	repo := new(mockBuyRepository)
	service := NewBuyService(repo)

	repo.On("GetAssetStatus", mock.Anything, int64(1)).Return(false, false, nil)

	err := service.MarkAssetSold(context.Background(), 1)

	require.ErrorIs(t, err, ErrNotFound)
	repo.AssertExpectations(t)
}

func TestBuyService_MarkAssetSold_Success(t *testing.T) {
	repo := new(mockBuyRepository)
	service := NewBuyService(repo)

	repo.On("GetAssetStatus", mock.Anything, int64(1)).Return(false, true, nil)
	repo.On("MarkAssetSold", mock.Anything, int64(1)).Return(nil)

	err := service.MarkAssetSold(context.Background(), 1)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBuyService_MarkStartupSold_AlreadySold(t *testing.T) {
	repo := new(mockBuyRepository)
	service := NewBuyService(repo)

	repo.On("GetStartupStatus", mock.Anything, int64(2)).Return("sold", nil)

	err := service.MarkStartupSold(context.Background(), 2)

	require.ErrorIs(t, err, ErrAlreadySold)
	repo.AssertExpectations(t)
}

func TestBuyService_MarkStartupSold_Success(t *testing.T) {
	repo := new(mockBuyRepository)
	service := NewBuyService(repo)

	repo.On("GetStartupStatus", mock.Anything, int64(2)).Return("active", nil)
	repo.On("MarkStartupSold", mock.Anything, int64(2)).Return(nil)

	err := service.MarkStartupSold(context.Background(), 2)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBuyService_UnlistAsset(t *testing.T) {
	repo := new(mockBuyRepository)
	service := NewBuyService(repo)

	repo.On("UnlistAsset", mock.Anything, int64(3)).Return(nil)

	err := service.UnlistAsset(context.Background(), 3)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBuyService_UnlistStartup(t *testing.T) {
	repo := new(mockBuyRepository)
	service := NewBuyService(repo)

	repo.On("UnlistStartup", mock.Anything, int64(4)).Return(nil)

	err := service.UnlistStartup(context.Background(), 4)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}
