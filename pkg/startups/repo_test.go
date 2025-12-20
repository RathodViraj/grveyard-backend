package startups

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func setupTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL_FOR_TEST")
	if dsn == "" {
		t.Skip("DATABASE_URL_FOR_TEST not set; skipping repository tests")
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

func cleanDatabase(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	_, err := pool.Exec(ctx, "TRUNCATE TABLE messages, chats, assets, startups, users RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func insertTestUserUUID(t *testing.T, pool *pgxpool.Pool, name string) string {
	t.Helper()

	ctx := context.Background()
	email := fmt.Sprintf("%s-%d@example.com", name, time.Now().UnixNano())
	userUUID := fmt.Sprintf("test-uuid-%d", time.Now().UnixNano())

	_, err := pool.Exec(ctx, "INSERT INTO users (name, email, role, password_hash, uuid) VALUES ($1, $2, 'founder', $3, $4)", name, email, "hash", userUUID)
	require.NoError(t, err)

	return userUUID
}

func TestPostgresStartupRepository_CreateStartup(t *testing.T) {
	pool := setupTestPool(t)
	// cleanDatabase(t, pool)

	repo := NewPostgresStartupRepository(pool)
	ctx := context.Background()
	ownerUUID := insertTestUserUUID(t, pool, "Alice")

	created, err := repo.CreateStartup(ctx, Startup{
		Name:        "Acme",
		Description: "Acme desc",
		LogoURL:     "https://example.com/logo.png",
		OwnerUUID:   ownerUUID,
		Status:      "active",
	})

	require.NoError(t, err)
	require.NotZero(t, created.ID)
	require.Equal(t, "Acme", created.Name)
}

func TestPostgresStartupRepository_UpdateStartup(t *testing.T) {
	pool := setupTestPool(t)
	// cleanDatabase(t, pool)

	repo := NewPostgresStartupRepository(pool)
	ctx := context.Background()
	ownerUUID := insertTestUserUUID(t, pool, "Bob")

	created, err := repo.CreateStartup(ctx, Startup{
		Name:        "Old",
		Description: "Old desc",
		LogoURL:     "old.png",
		OwnerUUID:   ownerUUID,
		Status:      "failed",
	})
	require.NoError(t, err)

	updated, err := repo.UpdateStartup(ctx, Startup{
		ID:          created.ID,
		Name:        "New",
		Description: "Updated desc",
		LogoURL:     "new.png",
		Status:      "sold",
	})

	require.NoError(t, err)
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, ownerUUID, updated.OwnerUUID)
	require.Equal(t, "New", updated.Name)
	require.Equal(t, "Updated desc", updated.Description)
	require.Equal(t, "new.png", updated.LogoURL)
	require.Equal(t, "sold", updated.Status)
}

func TestPostgresStartupRepository_DeleteStartup(t *testing.T) {
	pool := setupTestPool(t)
	// cleanDatabase(t, pool)

	repo := NewPostgresStartupRepository(pool)
	ctx := context.Background()
	ownerUUID := insertTestUserUUID(t, pool, "Carol")

	created, err := repo.CreateStartup(ctx, Startup{
		Name:        "DeleteMe",
		Description: "To be deleted",
		LogoURL:     "del.png",
		OwnerUUID:   ownerUUID,
		Status:      "active",
	})
	require.NoError(t, err)

	require.NoError(t, repo.DeleteStartup(ctx, created.ID))

	_, err = repo.GetStartupByID(ctx, created.ID)
	require.ErrorIs(t, err, ErrStartupNotFound)
}

// func TestPostgresStartupRepository_ListStartups(t *testing.T) {
// 	pool := setupTestPool(t)
// 	// cleanDatabase(t, pool)

// 	repo := NewPostgresStartupRepository(pool)
// 	ctx := context.Background()
// 	ownerID := insertTestUser(t, pool, "Dave")

// 	startupsToCreate := []Startup{
// 		{Name: "First", Description: "one", LogoURL: "1.png", OwnerID: ownerID, Status: "active"},
// 		{Name: "Second", Description: "two", LogoURL: "2.png", OwnerID: ownerID, Status: "failed"},
// 		{Name: "Third", Description: "three", LogoURL: "3.png", OwnerID: ownerID, Status: "sold"},
// 	}

// 	for _, s := range startupsToCreate {
// 		_, err := repo.CreateStartup(ctx, s)
// 		require.NoError(t, err)
// 	}

// 	items, _, err := repo.ListStartups(ctx, 2, 0)

// 	require.NoError(t, err)
// 	// require.EqualValues(t, 3, total)
// 	require.Len(t, items, 2)
// 	require.Equal(t, "First", items[0].Name)
// 	require.Equal(t, "Second", items[1].Name)
// }

// func TestPostgresStartupRepository_CreateStartup_InvalidOwner(t *testing.T) {
// 	pool := setupTestPool(t)
// 	// cleanDatabase(t, pool)

// 	repo := NewPostgresStartupRepository(pool)
// 	ctx := context.Background()

// 	_, err := repo.CreateStartup(ctx, Startup{
// 		Name:        "NoOwner",
// 		Description: "aaaaaaaaaaa",
// 		LogoURL:     "logo.png",
// 		OwnerID:     99999,
// 		Status:      "active",
// 	})

// 	require.Error(t, err)
// 	var pgErr *pgconn.PgError
// 	require.ErrorAs(t, err, &pgErr)
// 	require.Equal(t, "23503", pgErr.Code)
// }
