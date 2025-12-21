package assets

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAssetNotFound = errors.New("asset not found")

type AssetRepository interface {
	CreateAsset(ctx context.Context, input Asset) (Asset, error)
	UpdateAsset(ctx context.Context, input Asset) (Asset, error)
	DeleteAsset(ctx context.Context, id int64) error
	DeleteAllAssets(ctx context.Context) error
	DeleteAllAssetsByUserUUID(ctx context.Context, userUUID string) error
	GetAssetByID(ctx context.Context, id int64) (Asset, error)
	ListAssets(ctx context.Context, filters AssetFilters, limit, offset int) ([]Asset, int64, error)
	ListAssetsByUser(ctx context.Context, userUUID string, limit, offset int) ([]Asset, int64, error)
}

type AssetFilters struct {
	UserUUID  *string
	AssetType *string
	IsSold    *bool
}

type postgresAssetRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAssetRepository(pool *pgxpool.Pool) AssetRepository {
	return &postgresAssetRepository{pool: pool}
}

func (r *postgresAssetRepository) CreateAsset(ctx context.Context, input Asset) (Asset, error) {
	query := `INSERT INTO assets (user_uuid, title, description, asset_type, image_url, price, is_negotiable, is_sold, is_active, created_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
			  RETURNING id, user_uuid, title, description, asset_type, image_url, price, is_negotiable, is_sold, is_active, created_at`

	row := r.pool.QueryRow(ctx, query, input.UserUUID, input.Title, input.Description, input.AssetType, input.ImageURL, input.Price, input.IsNegotiable, input.IsSold, input.IsActive)

	var created Asset
	if err := row.Scan(&created.ID, &created.UserUUID, &created.Title, &created.Description, &created.AssetType, &created.ImageURL, &created.Price, &created.IsNegotiable, &created.IsSold, &created.IsActive, &created.CreatedAt); err != nil {
		return Asset{}, err
	}

	return created, nil
}

func (r *postgresAssetRepository) UpdateAsset(ctx context.Context, input Asset) (Asset, error) {
	query := `UPDATE assets
              SET title = $1, description = $2, asset_type = $3, image_url = $4, price = $5, is_negotiable = $6, is_sold = $7
              WHERE id = $8
			  RETURNING id, user_uuid, title, description, asset_type, image_url, price, is_negotiable, is_sold, is_active, created_at`

	row := r.pool.QueryRow(ctx, query, input.Title, input.Description, input.AssetType, input.ImageURL, input.Price, input.IsNegotiable, input.IsSold, input.ID)

	var updated Asset
	if err := row.Scan(&updated.ID, &updated.UserUUID, &updated.Title, &updated.Description, &updated.AssetType, &updated.ImageURL, &updated.Price, &updated.IsNegotiable, &updated.IsSold, &updated.IsActive, &updated.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Asset{}, ErrAssetNotFound
		}
		return Asset{}, err
	}

	return updated, nil
}

func (r *postgresAssetRepository) DeleteAsset(ctx context.Context, id int64) error {
	cmd, err := r.pool.Exec(ctx, "UPDATE assets SET is_deleted = true WHERE id = $1 AND is_deleted = false", id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrAssetNotFound
	}
	return nil
}

func (r *postgresAssetRepository) GetAssetByID(ctx context.Context, id int64) (Asset, error) {
	query := `SELECT id, user_uuid, title, description, asset_type, image_url, price, is_negotiable, is_sold, is_active, created_at
              FROM assets
              WHERE id = $1 AND is_deleted = false`

	row := r.pool.QueryRow(ctx, query, id)

	var a Asset
	if err := row.Scan(&a.ID, &a.UserUUID, &a.Title, &a.Description, &a.AssetType, &a.ImageURL, &a.Price, &a.IsNegotiable, &a.IsSold, &a.IsActive, &a.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Asset{}, ErrAssetNotFound
		}
		return Asset{}, err
	}

	return a, nil
}

func (r *postgresAssetRepository) ListAssets(ctx context.Context, filters AssetFilters, limit, offset int) ([]Asset, int64, error) {
	whereClauses := []string{"is_active = true", "is_deleted = false"}
	args := []interface{}{}
	argPos := 1

	if filters.UserUUID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("user_uuid = $%d", argPos))
		args = append(args, *filters.UserUUID)
		argPos++
	}

	if filters.AssetType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("asset_type = $%d", argPos))
		args = append(args, *filters.AssetType)
		argPos++
	}

	if filters.IsSold != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("is_sold = $%d", argPos))
		args = append(args, *filters.IsSold)
		argPos++
	}

	whereSQL := "WHERE " + strings.Join(whereClauses, " AND ")

	query := fmt.Sprintf(`SELECT id, user_uuid, title, description, asset_type, image_url, price, is_negotiable, is_sold, is_active, created_at
              FROM assets
              %s
              ORDER BY id
              LIMIT $%d OFFSET $%d`, whereSQL, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	assetsList := make([]Asset, 0)
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.UserUUID, &a.Title, &a.Description, &a.AssetType, &a.ImageURL, &a.Price, &a.IsNegotiable, &a.IsSold, &a.IsActive, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		assetsList = append(assetsList, a)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM assets %s", whereSQL)
	countArgs := args[:len(args)-2]

	var total int64
	countRow := r.pool.QueryRow(ctx, countQuery, countArgs...)
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, err
	}

	return assetsList, total, nil
}

func (r *postgresAssetRepository) ListAssetsByUser(ctx context.Context, userUUID string, limit, offset int) ([]Asset, int64, error) {
	query := `SELECT id, user_uuid, title, description, asset_type, image_url, price, is_negotiable, is_sold, is_active, created_at
              FROM assets
			  WHERE user_uuid = $1 AND is_active = true AND is_deleted = false
              ORDER BY id
              LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userUUID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	assetsList := make([]Asset, 0)
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.UserUUID, &a.Title, &a.Description, &a.AssetType, &a.ImageURL, &a.Price, &a.IsNegotiable, &a.IsSold, &a.IsActive, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		assetsList = append(assetsList, a)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int64
	countRow := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM assets WHERE user_uuid = $1 AND is_active = true AND is_deleted = false", userUUID)
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, err
	}

	return assetsList, total, nil
}

func (r *postgresAssetRepository) DeleteAllAssets(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, "UPDATE assets SET is_deleted = true WHERE is_deleted = false")
	return err
}

func (r *postgresAssetRepository) DeleteAllAssetsByUserUUID(ctx context.Context, userUUID string) error {
	_, err := r.pool.Exec(ctx, "UPDATE assets SET is_deleted = true WHERE user_uuid = $1 AND is_deleted = false", userUUID)
	return err
}
