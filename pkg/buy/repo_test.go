package buy

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"grveyard/pkg/testhelpers"
)

func setupBuyTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL_FOR_TEST")
	if dsn == "" {
		t.Skip("DATABASE_URL_FOR_TEST not set; skipping buy repository tests")
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

func cleanBuyTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, "TRUNCATE TABLE messages, chats, assets, startups, users RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func TestPostgresBuyRepository_MarkAssetSold(t *testing.T) {
	pool := setupBuyTestPool(t)
	// cleanBuyTables(t, pool)

	repo := NewPostgresBuyRepository(pool)
	ctx := context.Background()
	ownerID := testhelpers.CreateTestUser(t, pool)
	sid := testhelpers.CreateTestStartup(t, pool, ownerID)
	aid := testhelpers.CreateTestAsset(t, pool, sid)

	require.NoError(t, repo.MarkAssetSold(ctx, int64(aid)))

	sold, active, err := repo.GetAssetStatus(ctx, int64(aid))
	require.NoError(t, err)
	require.True(t, sold)
	require.True(t, active)
}

func TestPostgresBuyRepository_UnlistAsset(t *testing.T) {
	pool := setupBuyTestPool(t)
	// cleanBuyTables(t, pool)

	repo := NewPostgresBuyRepository(pool)
	ctx := context.Background()
	ownerID := testhelpers.CreateTestUser(t, pool)
	sid := testhelpers.CreateTestStartup(t, pool, ownerID)
	aid := testhelpers.CreateTestAsset(t, pool, sid)

	require.NoError(t, repo.UnlistAsset(ctx, int64(aid)))

	_, active, err := repo.GetAssetStatus(ctx, int64(aid))
	require.NoError(t, err)
	require.False(t, active)
}

func TestPostgresBuyRepository_MarkStartupSold(t *testing.T) {
	pool := setupBuyTestPool(t)
	// cleanBuyTables(t, pool)

	repo := NewPostgresBuyRepository(pool)
	ctx := context.Background()
	ownerID := testhelpers.CreateTestUser(t, pool)
	sid := testhelpers.CreateTestStartup(t, pool, ownerID)

	require.NoError(t, repo.MarkStartupSold(ctx, int64(sid)))

	status, err := repo.GetStartupStatus(ctx, int64(sid))
	require.NoError(t, err)
	require.Equal(t, "sold", status)
}

func TestPostgresBuyRepository_UnlistStartup(t *testing.T) {
	pool := setupBuyTestPool(t)
	// cleanBuyTables(t, pool)

	repo := NewPostgresBuyRepository(pool)
	ctx := context.Background()
	ownerID := testhelpers.CreateTestUser(t, pool)
	sid := testhelpers.CreateTestStartup(t, pool, ownerID)

	require.NoError(t, repo.UnlistStartup(ctx, int64(sid)))

	status, err := repo.GetStartupStatus(ctx, int64(sid))
	require.NoError(t, err)
	require.Equal(t, "failed", status)
}

func TestPostgresBuyRepository_GetAssetStatus_NotFound(t *testing.T) {
	pool := setupBuyTestPool(t)
	// cleanBuyTables(t, pool)

	repo := NewPostgresBuyRepository(pool)
	ctx := context.Background()

	_, _, err := repo.GetAssetStatus(ctx, 999)

	require.ErrorIs(t, err, ErrNotFound)
}

func TestPostgresBuyRepository_GetStartupStatus_NotFound(t *testing.T) {
	pool := setupBuyTestPool(t)
	// cleanBuyTables(t, pool)

	repo := NewPostgresBuyRepository(pool)
	ctx := context.Background()

	_, err := repo.GetStartupStatus(ctx, 999)

	require.ErrorIs(t, err, ErrNotFound)
}
