package assets

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAssetRepository struct {
	mock.Mock
}

func (m *mockAssetRepository) CreateAsset(ctx context.Context, input Asset) (Asset, error) {
	args := m.Called(ctx, input)
	asset, _ := args.Get(0).(Asset)
	return asset, args.Error(1)
}

func (m *mockAssetRepository) UpdateAsset(ctx context.Context, input Asset) (Asset, error) {
	args := m.Called(ctx, input)
	asset, _ := args.Get(0).(Asset)
	return asset, args.Error(1)
}

func (m *mockAssetRepository) DeleteAsset(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockAssetRepository) GetAssetByID(ctx context.Context, id int64) (Asset, error) {
	args := m.Called(ctx, id)
	asset, _ := args.Get(0).(Asset)
	return asset, args.Error(1)
}

func (m *mockAssetRepository) ListAssets(ctx context.Context, filters AssetFilters, limit, offset int) ([]Asset, int64, error) {
	args := m.Called(ctx, filters, limit, offset)
	assets, _ := args.Get(0).([]Asset)
	return assets, args.Get(1).(int64), args.Error(2)
}

func (m *mockAssetRepository) ListAssetsByStartup(ctx context.Context, startupID int64, limit, offset int) ([]Asset, int64, error) {
	args := m.Called(ctx, startupID, limit, offset)
	assets, _ := args.Get(0).([]Asset)
	return assets, args.Get(1).(int64), args.Error(2)
}

func TestAssetService_ListAssets_Defaults(t *testing.T) {
	repo := new(mockAssetRepository)
	service := NewAssetService(repo)

	repo.On("ListAssets", mock.Anything, AssetFilters{}, 10, 0).Return([]Asset{}, int64(0), nil)

	_, _, err := service.ListAssets(context.Background(), AssetFilters{}, 0, 0)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAssetService_ListAssetsByStartup_Defaults(t *testing.T) {
	repo := new(mockAssetRepository)
	service := NewAssetService(repo)

	repo.On("ListAssetsByStartup", mock.Anything, int64(5), 10, 0).Return([]Asset{}, int64(0), nil)

	_, _, err := service.ListAssetsByStartup(context.Background(), 5, 0, 0)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAssetService_CreateAsset_Delegates(t *testing.T) {
	repo := new(mockAssetRepository)
	service := NewAssetService(repo)

	expected := Asset{ID: 1, Title: "A"}
	repo.On("CreateAsset", mock.Anything, expected).Return(expected, nil)

	got, err := service.CreateAsset(context.Background(), expected)

	require.NoError(t, err)
	require.Equal(t, expected, got)
	repo.AssertExpectations(t)
}
