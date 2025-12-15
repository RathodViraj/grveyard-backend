package startups

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrStartupNotFound = errors.New("startup not found")

type StartupRepository interface {
	CreateStartup(ctx context.Context, input Startup) (Startup, error)
	UpdateStartup(ctx context.Context, input Startup) (Startup, error)
	DeleteStartup(ctx context.Context, id int64) error
	GetStartupByID(ctx context.Context, id int64) (Startup, error)
	ListStartups(ctx context.Context, limit, offset int) ([]Startup, int64, error)
}

type postgresStartupRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresStartupRepository(pool *pgxpool.Pool) StartupRepository {
	return &postgresStartupRepository{pool: pool}
}

func (r *postgresStartupRepository) CreateStartup(ctx context.Context, input Startup) (Startup, error) {
	query := `INSERT INTO startups (name, description, logo_url, owner_id, status, created_at)
              VALUES ($1, $2, $3, $4, $5, NOW())
              RETURNING id, name, description, logo_url, owner_id, status, created_at`

	row := r.pool.QueryRow(ctx, query, input.Name, input.Description, input.LogoURL, input.OwnerID, input.Status)

	var created Startup
	if err := row.Scan(&created.ID, &created.Name, &created.Description, &created.LogoURL, &created.OwnerID, &created.Status, &created.CreatedAt); err != nil {
		return Startup{}, err
	}

	return created, nil
}

func (r *postgresStartupRepository) UpdateStartup(ctx context.Context, input Startup) (Startup, error) {
	query := `UPDATE startups
              SET name = $1, description = $2, logo_url = $3, status = $4
              WHERE id = $5
              RETURNING id, name, description, logo_url, owner_id, status, created_at`

	row := r.pool.QueryRow(ctx, query, input.Name, input.Description, input.LogoURL, input.Status, input.ID)

	var updated Startup
	if err := row.Scan(&updated.ID, &updated.Name, &updated.Description, &updated.LogoURL, &updated.OwnerID, &updated.Status, &updated.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Startup{}, ErrStartupNotFound
		}
		return Startup{}, err
	}

	return updated, nil
}

func (r *postgresStartupRepository) DeleteStartup(ctx context.Context, id int64) error {
	cmd, err := r.pool.Exec(ctx, "UPDATE startups SET is_deleted = true WHERE id = $1 AND is_deleted = false", id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrStartupNotFound
	}
	return nil
}

func (r *postgresStartupRepository) GetStartupByID(ctx context.Context, id int64) (Startup, error) {
	query := `SELECT id, name, description, logo_url, owner_id, status, created_at
              FROM startups
              WHERE id = $1 AND is_deleted = false`

	row := r.pool.QueryRow(ctx, query, id)

	var s Startup
	if err := row.Scan(&s.ID, &s.Name, &s.Description, &s.LogoURL, &s.OwnerID, &s.Status, &s.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Startup{}, ErrStartupNotFound
		}
		return Startup{}, err
	}

	return s, nil
}

func (r *postgresStartupRepository) ListStartups(ctx context.Context, limit, offset int) ([]Startup, int64, error) {
	query := `SELECT id, name, description, logo_url, owner_id, status, created_at
              FROM startups
              WHERE is_deleted = false
              ORDER BY id
              LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	startups := make([]Startup, 0)
	for rows.Next() {
		var s Startup
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.LogoURL, &s.OwnerID, &s.Status, &s.CreatedAt); err != nil {
			return nil, 0, err
		}
		startups = append(startups, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int64
	countRow := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM startups WHERE is_deleted = false")
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, err
	}

	return startups, total, nil
}
