package buy

import "context"

type BuyService interface {
	MarkAssetSold(ctx context.Context, assetID int64) error
	UnlistAsset(ctx context.Context, assetID int64) error
	MarkStartupSold(ctx context.Context, startupID int64) error
	UnlistStartup(ctx context.Context, startupID int64) error
}

type buyService struct {
	repo BuyRepository
}

func NewBuyService(repo BuyRepository) BuyService {
	return &buyService{repo: repo}
}

func (s *buyService) MarkAssetSold(ctx context.Context, assetID int64) error {
	isSold, isActive, err := s.repo.GetAssetStatus(ctx, assetID)
	if err != nil {
		return err
	}

	if !isActive {
		return ErrNotFound
	}

	if isSold {
		return ErrAlreadySold
	}

	return s.repo.MarkAssetSold(ctx, assetID)
}

func (s *buyService) UnlistAsset(ctx context.Context, assetID int64) error {
	return s.repo.UnlistAsset(ctx, assetID)
}

func (s *buyService) MarkStartupSold(ctx context.Context, startupID int64) error {
	status, err := s.repo.GetStartupStatus(ctx, startupID)
	if err != nil {
		return err
	}

	if status == "sold" {
		return ErrAlreadySold
	}

	return s.repo.MarkStartupSold(ctx, startupID)
}

func (s *buyService) UnlistStartup(ctx context.Context, startupID int64) error {
	return s.repo.UnlistStartup(ctx, startupID)
}
