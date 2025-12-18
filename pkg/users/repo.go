package users

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

//go:generate mockgen -destination=./mock_users_repo.go -package=users . UserRepository

type UserRepository interface {
	CreateUser(ctx context.Context, name, email, role, passwordHash, profilePicURL, uuid string) (User, error)
	UpdateUser(ctx context.Context, u User) (User, error)
	UpdateUserByUUID(ctx context.Context, currentUUID string, u User) (User, error)
	DeleteUser(ctx context.Context, id int64) error
	DeleteUserByUUID(ctx context.Context, uuid string) error
	GetUserByID(ctx context.Context, id int64) (User, error)
	GetUserByUUID(ctx context.Context, uuid string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByEmailIncludingDeleted(ctx context.Context, email string) (User, error)
	ReviveUserByEmail(ctx context.Context, email, name, role, passwordHash, profilePicURL, uuid string) (User, error)
	ListUsers(ctx context.Context, limit, offset int) ([]User, int64, error)
	// Auth helpers
	GetUserAuthByEmail(ctx context.Context, email string) (int64, string, error)
	UpdateVerifiedAtByEmail(ctx context.Context, email string, ts time.Time) error
}

type postgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) UserRepository {
	return &postgresUserRepository{pool: pool}
}

func (r *postgresUserRepository) CreateUser(ctx context.Context, name, email, role, passwordHash, profilePicURL, uuid string) (User, error) {
	query := `INSERT INTO users (name, email, role, password_hash, profile_pic_url, uuid, created_at)
              VALUES ($1, $2, $3, $4, $5, $6, NOW())
              RETURNING id, name, email, role, profile_pic_url, uuid, verified_at, created_at`
	row := r.pool.QueryRow(ctx, query, name, email, role, passwordHash, profilePicURL, uuid)

	var u User
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.ProfilePicURL, &u.UUID, &u.VerifiedAt, &u.CreatedAt); err != nil {
		return User{}, err
	}
	return u, nil
}

func (r *postgresUserRepository) UpdateUser(ctx context.Context, u User) (User, error) {
	query := `UPDATE users
              SET name = $1, role = $2, profile_pic_url = $3, uuid = $4
              WHERE id = $5 AND is_deleted = false
              RETURNING id, name, email, role, profile_pic_url, uuid, verified_at, created_at`
	row := r.pool.QueryRow(ctx, query, u.Name, u.Role, u.ProfilePicURL, u.UUID, u.ID)

	var out User
	if err := row.Scan(&out.ID, &out.Name, &out.Email, &out.Role, &out.ProfilePicURL, &out.UUID, &out.VerifiedAt, &out.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return out, nil
}

func (r *postgresUserRepository) UpdateUserByUUID(ctx context.Context, currentUUID string, u User) (User, error) {
	query := `UPDATE users
			  SET name = $1, role = $2, profile_pic_url = $3, uuid = $4
			  WHERE uuid = $5 AND is_deleted = false
              RETURNING id, name, email, role, profile_pic_url, uuid, verified_at, created_at`
	row := r.pool.QueryRow(ctx, query, u.Name, u.Role, u.ProfilePicURL, u.UUID, currentUUID)

	var out User
	if err := row.Scan(&out.ID, &out.Name, &out.Email, &out.Role, &out.ProfilePicURL, &out.UUID, &out.VerifiedAt, &out.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return out, nil
}

func (r *postgresUserRepository) DeleteUser(ctx context.Context, id int64) error {
	cmd, err := r.pool.Exec(ctx, "UPDATE users SET email = NULL, is_deleted = true WHERE id = $1 AND is_deleted = false", id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *postgresUserRepository) DeleteUserByUUID(ctx context.Context, uuid string) error {
	cmd, err := r.pool.Exec(ctx, "UPDATE users SET email = NULL, is_deleted = true WHERE uuid = $1 AND is_deleted = false", uuid)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *postgresUserRepository) GetUserByEmailIncludingDeleted(ctx context.Context, email string) (User, error) {
	query := `SELECT id, name, email, role, profile_pic_url, uuid, verified_at, created_at, is_deleted
			  FROM users
			  WHERE email = $1`
	row := r.pool.QueryRow(ctx, query, email)

	var u User
	var isDeleted bool
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.ProfilePicURL, &u.UUID, &u.VerifiedAt, &u.CreatedAt, &isDeleted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	if isDeleted {
		// keep info; caller can decide to revive
	}
	return u, nil
}

func (r *postgresUserRepository) ReviveUserByEmail(ctx context.Context, email, name, role, passwordHash, profilePicURL, uuid string) (User, error) {
	query := `UPDATE users
			  SET name = $1, role = $2, password_hash = $3, profile_pic_url = $4, uuid = $5, is_deleted = false
			  WHERE email = $6
			  RETURNING id, name, email, role, profile_pic_url, uuid, verified_at, created_at`
	row := r.pool.QueryRow(ctx, query, name, role, passwordHash, profilePicURL, uuid, email)

	var u User
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.ProfilePicURL, &u.UUID, &u.VerifiedAt, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (r *postgresUserRepository) GetUserByID(ctx context.Context, id int64) (User, error) {
	query := `SELECT id, name, email, role, profile_pic_url, uuid, verified_at, created_at
              FROM users
              WHERE id = $1 AND is_deleted = false`
	row := r.pool.QueryRow(ctx, query, id)

	var u User
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.ProfilePicURL, &u.UUID, &u.VerifiedAt, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (r *postgresUserRepository) GetUserByUUID(ctx context.Context, uuid string) (User, error) {
	query := `SELECT id, name, email, role, profile_pic_url, uuid, verified_at, created_at
			  FROM users
			  WHERE uuid = $1 AND is_deleted = false`
	row := r.pool.QueryRow(ctx, query, uuid)

	var u User
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.ProfilePicURL, &u.UUID, &u.VerifiedAt, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (r *postgresUserRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	query := `SELECT id, name, email, role, profile_pic_url, uuid, verified_at, created_at
			  FROM users
			  WHERE email = $1 AND is_deleted = false`
	row := r.pool.QueryRow(ctx, query, email)

	var u User
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.ProfilePicURL, &u.UUID, &u.VerifiedAt, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (r *postgresUserRepository) ListUsers(ctx context.Context, limit, offset int) ([]User, int64, error) {
	query := `SELECT id, name, email, role, profile_pic_url, uuid, verified_at, created_at
              FROM users
              WHERE is_deleted = false
              ORDER BY id
              LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.ProfilePicURL, &u.UUID, &u.VerifiedAt, &u.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int64
	countRow := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE is_deleted = false")
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *postgresUserRepository) GetUserAuthByEmail(ctx context.Context, email string) (int64, string, error) {
	var id int64
	var hash string
	row := r.pool.QueryRow(ctx, `SELECT id, password_hash FROM users WHERE email = $1 AND is_deleted = false`, email)
	if err := row.Scan(&id, &hash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", ErrUserNotFound
		}
		return 0, "", err
	}
	return id, hash, nil
}

func (r *postgresUserRepository) UpdateVerifiedAtByEmail(ctx context.Context, email string, ts time.Time) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE users SET verified_at = $1 WHERE email = $2 AND is_deleted = false`, ts, email)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// Removed UpdateUserUUID: login no longer changes UUID
