package users

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func setupUserTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL_FOR_TEST")
	if dsn == "" {
		t.Skip("DATABASE_URL_FOR_TEST not set; skipping user repository tests")
	}

	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	require.NoError(t, pool.Ping(ctx))

	t.Cleanup(pool.Close)
	return pool
}

func cleanUserTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	_, err := pool.Exec(ctx, "TRUNCATE TABLE messages, chats, assets, startups, users, otps RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func insertUser(t *testing.T, pool *pgxpool.Pool, name string) User {
	t.Helper()

	ctx := context.Background()
	email := fmt.Sprintf("%s-%d@example.com", name, time.Now().UnixNano())
	repo := NewPostgresUserRepository(pool)
	created, err := repo.CreateUser(ctx, name, email, "buyer", "hash", "", fmt.Sprintf("uuid-%d", time.Now().UnixNano()))
	require.NoError(t, err)
	return created
}

func TestPostgresUserRepository_CreateUser(t *testing.T) {
	pool := setupUserTestPool(t)
	//cleanUserTables(t, pool)

	repo := NewPostgresUserRepository(pool)
	ctx := context.Background()
	email := fmt.Sprintf("user-%d@example.com", time.Now().UnixNano())

	created, err := repo.CreateUser(ctx, "Alice", email, "buyer", "hash", "pic.png", "uuid-1")

	require.NoError(t, err)
	require.NotZero(t, created.ID)
	require.Equal(t, "Alice", created.Name)
	require.Equal(t, email, created.Email)
	require.Equal(t, "buyer", created.Role)
	require.False(t, created.CreatedAt.IsZero())
}

func TestPostgresUserRepository_UpdateUser(t *testing.T) {
	pool := setupUserTestPool(t)
	// cleanUserTables(t, pool)

	repo := NewPostgresUserRepository(pool)
	ctx := context.Background()
	created := insertUser(t, pool, "Bob")

	updated, err := repo.UpdateUser(ctx, User{
		ID:            created.ID,
		Name:          "Bobby",
		Role:          "founder",
		ProfilePicURL: "new-pic.png",
		UUID:          "uuid-updated",
	})

	require.NoError(t, err)
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, "Bobby", updated.Name)
	require.Equal(t, "founder", updated.Role)
	require.Equal(t, "new-pic.png", updated.ProfilePicURL)
	require.Equal(t, "uuid-updated", updated.UUID)
}

func TestPostgresUserRepository_DeleteUser(t *testing.T) {
	pool := setupUserTestPool(t)
	// cleanUserTables(t, pool)

	repo := NewPostgresUserRepository(pool)
	ctx := context.Background()
	created := insertUser(t, pool, "Carol")

	require.NoError(t, repo.DeleteUser(ctx, created.ID))

	_, err := repo.GetUserByID(ctx, created.ID)
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestPostgresUserRepository_ListUsers(t *testing.T) {
	pool := setupUserTestPool(t)
	cleanUserTables(t, pool)

	repo := NewPostgresUserRepository(pool)
	ctx := context.Background()
	insertUser(t, pool, "First")
	insertUser(t, pool, "Second")
	insertUser(t, pool, "Third")

	users, total, err := repo.ListUsers(ctx, 2, 0)

	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, users, 2)
	require.Equal(t, "First", users[0].Name)
	require.Equal(t, "Second", users[1].Name)
}

func TestPostgresUserRepository_UpdateUser_NotFound(t *testing.T) {
	pool := setupUserTestPool(t)
	// cleanUserTables(t, pool)

	repo := NewPostgresUserRepository(pool)
	ctx := context.Background()

	_, err := repo.UpdateUser(ctx, User{ID: 999, Name: "Ghost", Role: "buyer"})

	require.ErrorIs(t, err, ErrUserNotFound)
}
