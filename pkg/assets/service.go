package assets

import "context"

type AssetService interface {
	CreateAsset(ctx context.Context, input Asset) (Asset, error)
	UpdateAsset(ctx context.Context, input Asset) (Asset, error)
	DeleteAsset(ctx context.Context, id int64) error
	DeleteAllAssets(ctx context.Context) error
	DeleteAllAssetsByUserUUID(ctx context.Context, userUUID string) error
	GetAssetByID(ctx context.Context, id int64) (Asset, error)
	ListAssets(ctx context.Context, filters AssetFilters, page, limit int) ([]Asset, int64, error)
	ListAssetsByUser(ctx context.Context, userUUID string, page, limit int) ([]Asset, int64, error)
}

type assetService struct {
	repo AssetRepository
}

func NewAssetService(repo AssetRepository) AssetService {
	return &assetService{repo: repo}
}

func (s *assetService) CreateAsset(ctx context.Context, input Asset) (Asset, error) {
	return s.repo.CreateAsset(ctx, input)
}

func (s *assetService) UpdateAsset(ctx context.Context, input Asset) (Asset, error) {
	return s.repo.UpdateAsset(ctx, input)
}

func (s *assetService) DeleteAsset(ctx context.Context, id int64) error {
	return s.repo.DeleteAsset(ctx, id)
}

func (s *assetService) GetAssetByID(ctx context.Context, id int64) (Asset, error) {
	return s.repo.GetAssetByID(ctx, id)
}

func (s *assetService) ListAssets(ctx context.Context, filters AssetFilters, page, limit int) ([]Asset, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	return s.repo.ListAssets(ctx, filters, limit, offset)
}

func (s *assetService) ListAssetsByUser(ctx context.Context, userUUID string, page, limit int) ([]Asset, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	return s.repo.ListAssetsByUser(ctx, userUUID, limit, offset)
}

func (s *assetService) DeleteAllAssets(ctx context.Context) error {
	return s.repo.DeleteAllAssets(ctx)
}

func (s *assetService) DeleteAllAssetsByUserUUID(ctx context.Context, userUUID string) error {
	return s.repo.DeleteAllAssetsByUserUUID(ctx, userUUID)
}
