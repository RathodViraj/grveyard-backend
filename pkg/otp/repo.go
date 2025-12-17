package otp

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OTPRepository interface {
	CreateOTP(ctx context.Context, email, code string, expiresAt time.Time) (OTP, error)
	GetOTPByEmail(ctx context.Context, email string) (OTP, error)
	MarkOTPAsVerified(ctx context.Context, id int64) error
	DeleteExpiredOTPs(ctx context.Context) error
}

type postgresOTPRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresOTPRepository(pool *pgxpool.Pool) OTPRepository {
	return &postgresOTPRepository{pool: pool}
}

func (r *postgresOTPRepository) CreateOTP(ctx context.Context, email, code string, expiresAt time.Time) (OTP, error) {
	query := `
		INSERT INTO otps (email, code, expires_at, verified, created_at)
		VALUES ($1, $2, $3, false, NOW())
		RETURNING id, email, code, expires_at, verified, created_at
	`

	var otp OTP
	err := r.pool.QueryRow(ctx, query, email, code, expiresAt).Scan(
		&otp.ID,
		&otp.Email,
		&otp.Code,
		&otp.ExpiresAt,
		&otp.Verified,
		&otp.CreatedAt,
	)
	return otp, err
}

func (r *postgresOTPRepository) GetOTPByEmail(ctx context.Context, email string) (OTP, error) {
	query := `
		SELECT id, email, code, expires_at, verified, created_at
		FROM otps
		WHERE email = $1 AND verified = false
		ORDER BY created_at DESC
		LIMIT 1
	`

	var otp OTP
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&otp.ID,
		&otp.Email,
		&otp.Code,
		&otp.ExpiresAt,
		&otp.Verified,
		&otp.CreatedAt,
	)
	return otp, err
}

func (r *postgresOTPRepository) MarkOTPAsVerified(ctx context.Context, id int64) error {
	query := `UPDATE otps SET verified = true WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *postgresOTPRepository) DeleteExpiredOTPs(ctx context.Context) error {
	query := `DELETE FROM otps WHERE expires_at < NOW()`
	_, err := r.pool.Exec(ctx, query)
	return err
}
