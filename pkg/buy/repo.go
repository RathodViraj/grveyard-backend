package buy

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadySold   = errors.New("already marked as sold")
	ErrInvalidEntity = errors.New("invalid entity type")
)

type BuyRepository interface {
	MarkAssetSold(ctx context.Context, assetID int64) error
	UnlistAsset(ctx context.Context, assetID int64) error
	MarkStartupSold(ctx context.Context, startupID int64) error
	UnlistStartup(ctx context.Context, startupID int64) error
	GetAssetStatus(ctx context.Context, assetID int64) (bool, bool, error)
	GetStartupStatus(ctx context.Context, startupID int64) (string, error)
}

type postgresBuyRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresBuyRepository(pool *pgxpool.Pool) BuyRepository {
	return &postgresBuyRepository{pool: pool}
}

func (r *postgresBuyRepository) MarkAssetSold(ctx context.Context, assetID int64) error {
	query := `UPDATE assets SET is_sold = true WHERE id = $1 AND is_active = true`
	cmd, err := r.pool.Exec(ctx, query, assetID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresBuyRepository) UnlistAsset(ctx context.Context, assetID int64) error {
	query := `UPDATE assets SET is_active = false WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, assetID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresBuyRepository) MarkStartupSold(ctx context.Context, startupID int64) error {
	query := `UPDATE startups SET status = 'sold' WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, startupID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresBuyRepository) UnlistStartup(ctx context.Context, startupID int64) error {
	query := `UPDATE startups SET status = 'failed' WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, startupID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresBuyRepository) GetAssetStatus(ctx context.Context, assetID int64) (bool, bool, error) {
	query := `SELECT is_sold, is_active FROM assets WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, assetID)

	var isSold, isActive bool
	if err := row.Scan(&isSold, &isActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, ErrNotFound
		}
		return false, false, err
	}

	return isSold, isActive, nil
}

func (r *postgresBuyRepository) GetStartupStatus(ctx context.Context, startupID int64) (string, error) {
	query := `SELECT status FROM startups WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, startupID)

	var status string
	if err := row.Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}

	return status, nil
}
