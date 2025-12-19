package testhelpers

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var uniqueCounter int64

func nextSuffix() int64 {
	return atomic.AddInt64(&uniqueCounter, 1)
}

// CreateTestUser inserts a minimal valid user row and returns its ID.
func CreateTestUser(t *testing.T, db *pgxpool.Pool) int {
	t.Helper()

	ctx := context.Background()
	suffix := nextSuffix()
	name := fmt.Sprintf("test-user-%d", suffix)
	email := fmt.Sprintf("%s@example.com", name)

	var id int64
	err := db.QueryRow(ctx, "INSERT INTO users (name, email, role, password_hash) VALUES ($1, $2, 'founder', $3) RETURNING id", name, email, "hash").Scan(&id)
	require.NoError(t, err)
	return int(id)
}

// CreateTestStartup inserts a startup for the given owner and returns its ID.
func CreateTestStartup(t *testing.T, db *pgxpool.Pool, ownerID int) int {
	t.Helper()

	ctx := context.Background()
	suffix := nextSuffix()
	name := fmt.Sprintf("test-startup-%d", suffix)

	var id int64
	err := db.QueryRow(ctx, "INSERT INTO startups (name, owner_id, status) VALUES ($1, $2, 'active') RETURNING id", name, ownerID).Scan(&id)
	require.NoError(t, err)
	return int(id)
}

// CreateTestAsset inserts an active, unsold asset for the given startup and returns its ID.
func CreateTestAsset(t *testing.T, db *pgxpool.Pool, startupID int) int {
	t.Helper()

	ctx := context.Background()
	suffix := nextSuffix()
	title := fmt.Sprintf("test-asset-%d", suffix)

	var id int64
	err := db.QueryRow(ctx, "INSERT INTO assets (startup_id, title, asset_type) VALUES ($1, $2, 'research') RETURNING id", startupID, title).Scan(&id)
	require.NoError(t, err)
	return int(id)
}
