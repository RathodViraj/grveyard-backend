package assets

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"grveyard/pkg/testhelpers"
)

func setupAssetTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL_FOR_TEST")
	if dsn == "" {
		t.Skip("DATABASE_URL_FOR_TEST not set; skipping asset repository tests")
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

func cleanAssetTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, "TRUNCATE TABLE messages, chats, assets, startups, users RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func TestPostgresAssetRepository_CreateAsset(t *testing.T) {
	pool := setupAssetTestPool(t)
	// cleanAssetTables(t, pool)

	repo := NewPostgresAssetRepository(pool)
	ctx := context.Background()
	ownerID := testhelpers.CreateTestUser(t, pool)
	sid := int64(testhelpers.CreateTestStartup(t, pool, ownerID))

	created, err := repo.CreateAsset(ctx, Asset{
		StartupID:    sid,
		Title:        "Asset",
		Description:  "desc",
		AssetType:    "research",
		ImageURL:     "img",
		Price:        100,
		IsNegotiable: true,
		IsSold:       false,
		IsActive:     true,
	})

	require.NoError(t, err)
	require.NotZero(t, created.ID)
	require.Equal(t, sid, created.StartupID)
	require.Equal(t, "Asset", created.Title)
	require.False(t, created.CreatedAt.IsZero())
}

func TestPostgresAssetRepository_UpdateAsset(t *testing.T) {
	pool := setupAssetTestPool(t)
	// cleanAssetTables(t, pool)

	repo := NewPostgresAssetRepository(pool)
	ctx := context.Background()
	ownerID := testhelpers.CreateTestUser(t, pool)
	sid := int64(testhelpers.CreateTestStartup(t, pool, ownerID))

	created, err := repo.CreateAsset(ctx, Asset{StartupID: sid, Title: "Old", AssetType: "data", IsNegotiable: true})
	require.NoError(t, err)

	updated, err := repo.UpdateAsset(ctx, Asset{
		ID:           created.ID,
		Title:        "New",
		Description:  "updated",
		AssetType:    "product",
		ImageURL:     "img-new",
		Price:        50,
		IsNegotiable: false,
		IsSold:       true,
	})

	require.NoError(t, err)
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, "New", updated.Title)
	require.Equal(t, "product", updated.AssetType)
	require.True(t, updated.IsSold)
}

func TestPostgresAssetRepository_DeleteAsset(t *testing.T) {
	pool := setupAssetTestPool(t)
	// cleanAssetTables(t, pool)

	repo := NewPostgresAssetRepository(pool)
	ctx := context.Background()
	ownerID := testhelpers.CreateTestUser(t, pool)
	sid := int64(testhelpers.CreateTestStartup(t, pool, ownerID))

	created, err := repo.CreateAsset(ctx, Asset{StartupID: sid, Title: "Delete", AssetType: "domain"})
	require.NoError(t, err)

	require.NoError(t, repo.DeleteAsset(ctx, created.ID))

	_, err = repo.GetAssetByID(ctx, created.ID)
	require.ErrorIs(t, err, ErrAssetNotFound)
}

func TestPostgresAssetRepository_ListAssets_WithFilters(t *testing.T) {
	pool := setupAssetTestPool(t)
	// cleanAssetTables(t, pool)

	repo := NewPostgresAssetRepository(pool)
	ctx := context.Background()
	ownerID := testhelpers.CreateTestUser(t, pool)
	sid := int64(testhelpers.CreateTestStartup(t, pool, ownerID))

	assetsToCreate := []Asset{
		{StartupID: sid, Title: "One", AssetType: "research", IsSold: false, IsActive: true},
		{StartupID: sid, Title: "Two", AssetType: "product", IsSold: true, IsActive: true},
		{StartupID: sid, Title: "Three", AssetType: "product", IsSold: false, IsActive: true},
	}
	for _, a := range assetsToCreate {
		_, err := repo.CreateAsset(ctx, a)
		require.NoError(t, err)
	}

	filters := AssetFilters{AssetType: ptrString("product"), IsSold: ptrBool(false)}
	items, total, err := repo.ListAssets(ctx, filters, 10, 0)

	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, "Three", items[0].Title)
}

// func TestPostgresAssetRepository_ListAssets_Pagination(t *testing.T) {
// 	pool := setupAssetTestPool(t)
// 	// cleanAssetTables(t, pool)

// 	repo := NewPostgresAssetRepository(pool)
// 	ctx := context.Background()
// 	ownerID := testhelpers.CreateTestUser(t, pool)
// 	sid := int64(testhelpers.CreateTestStartup(t, pool, ownerID))

// 	for i := 0; i < 3; i++ {
// 		_, err := repo.CreateAsset(ctx, Asset{StartupID: sid, Title: fmt.Sprintf("A%d", i+1), AssetType: "research", IsActive: true})
// 		require.NoError(t, err)
// 	}

// 	items, _, err := repo.ListAssets(ctx, AssetFilters{}, 2, 0)

// 	require.NoError(t, err)
// 	// require.EqualValues(t, 3, total)
// 	require.Len(t, items, 2)
// 	require.Equal(t, "A1", items[0].Title)
// 	require.Equal(t, "A2", items[1].Title)
// }

// func TestPostgresAssetRepository_CreateAsset_InvalidStartup(t *testing.T) {
// 	pool := setupAssetTestPool(t)
// 	// cleanAssetTables(t, pool)

// 	repo := NewPostgresAssetRepository(pool)
// 	ctx := context.Background()

// 	_, err := repo.CreateAsset(ctx, Asset{StartupID: 9999, Title: "Bad", AssetType: "research"})

// 	require.Error(t, err)
// 	var pgErr *pgconn.PgError
// 	require.ErrorAs(t, err, &pgErr)
// 	require.Equal(t, "23503", pgErr.Code)
// }

func ptrString(v string) *string { return &v }
func ptrBool(v bool) *bool       { return &v }
