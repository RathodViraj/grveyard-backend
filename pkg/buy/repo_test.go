package buy

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func setupBuyTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping buy repository tests")
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

func insertStartupForBuy(t *testing.T, pool *pgxpool.Pool, name string) int64 {
	t.Helper()
	ctx := context.Background()
	email := fmt.Sprintf("%s-%d@example.com", name, time.Now().UnixNano())
	var userID int64
	err := pool.QueryRow(ctx, "INSERT INTO users (name, email, role, password_hash) VALUES ($1,$2,'founder',$3) RETURNING id", name, email, "hash").Scan(&userID)
	require.NoError(t, err)

	var startupID int64
	err = pool.QueryRow(ctx, "INSERT INTO startups (name, owner_id, status) VALUES ($1,$2,'active') RETURNING id", name+"-startup", userID).Scan(&startupID)
	require.NoError(t, err)
	return startupID
}

func insertAssetForBuy(t *testing.T, pool *pgxpool.Pool, startupID int64, title string) int64 {
	t.Helper()
	ctx := context.Background()
	var assetID int64
	err := pool.QueryRow(ctx, "INSERT INTO assets (startup_id, title, asset_type, is_active, is_sold, created_at) VALUES ($1,$2,'research',true,false,NOW()) RETURNING id", startupID, title).Scan(&assetID)
	require.NoError(t, err)
	return assetID
}

func TestPostgresBuyRepository_MarkAssetSold(t *testing.T) {
	pool := setupBuyTestPool(t)
	cleanBuyTables(t, pool)

	repo := NewPostgresBuyRepository(pool)
	ctx := context.Background()
	sid := insertStartupForBuy(t, pool, "Alice")
	aid := insertAssetForBuy(t, pool, sid, "Asset")

	require.NoError(t, repo.MarkAssetSold(ctx, aid))

	sold, active, err := repo.GetAssetStatus(ctx, aid)
	require.NoError(t, err)
	require.True(t, sold)
	require.True(t, active)
}

func TestPostgresBuyRepository_UnlistAsset(t *testing.T) {
	pool := setupBuyTestPool(t)
	cleanBuyTables(t, pool)

	repo := NewPostgresBuyRepository(pool)
	ctx := context.Background()
	sid := insertStartupForBuy(t, pool, "Bob")
	aid := insertAssetForBuy(t, pool, sid, "Asset")

	require.NoError(t, repo.UnlistAsset(ctx, aid))

	_, active, err := repo.GetAssetStatus(ctx, aid)
	require.NoError(t, err)
	require.False(t, active)
}

func TestPostgresBuyRepository_MarkStartupSold(t *testing.T) {
	pool := setupBuyTestPool(t)
	cleanBuyTables(t, pool)

	repo := NewPostgresBuyRepository(pool)
	ctx := context.Background()
	sid := insertStartupForBuy(t, pool, "Carol")

	require.NoError(t, repo.MarkStartupSold(ctx, sid))

	status, err := repo.GetStartupStatus(ctx, sid)
	require.NoError(t, err)
	require.Equal(t, "sold", status)
}

func TestPostgresBuyRepository_UnlistStartup(t *testing.T) {
	pool := setupBuyTestPool(t)
	cleanBuyTables(t, pool)

	repo := NewPostgresBuyRepository(pool)
	ctx := context.Background()
	sid := insertStartupForBuy(t, pool, "Dave")

	require.NoError(t, repo.UnlistStartup(ctx, sid))

	status, err := repo.GetStartupStatus(ctx, sid)
	require.NoError(t, err)
	require.Equal(t, "failed", status)
}

func TestPostgresBuyRepository_GetAssetStatus_NotFound(t *testing.T) {
	pool := setupBuyTestPool(t)
	cleanBuyTables(t, pool)

	repo := NewPostgresBuyRepository(pool)
	ctx := context.Background()

	_, _, err := repo.GetAssetStatus(ctx, 999)

	require.ErrorIs(t, err, ErrNotFound)
}

func TestPostgresBuyRepository_GetStartupStatus_NotFound(t *testing.T) {
	pool := setupBuyTestPool(t)
	cleanBuyTables(t, pool)

	repo := NewPostgresBuyRepository(pool)
	ctx := context.Background()

	_, err := repo.GetStartupStatus(ctx, 999)

	require.ErrorIs(t, err, ErrNotFound)
}
