package testhelpers

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var (
	uniqueCounter int64
	baseSuffix    = time.Now().UnixNano()
)

func nextSuffix() int64 {
	return baseSuffix + atomic.AddInt64(&uniqueCounter, 1)
}

// CreateTestUser inserts a minimal valid user row and returns its UUID.
func CreateTestUser(t *testing.T, db *pgxpool.Pool) string {
	t.Helper()

	ctx := context.Background()
	suffix := nextSuffix()
	name := fmt.Sprintf("test-user-%d", suffix)
	email := fmt.Sprintf("%s@example.com", name)
	uuid := fmt.Sprintf("uuid-%d", suffix)

	var outUUID string
	err := db.QueryRow(ctx, "INSERT INTO users (name, email, role, password_hash, uuid) VALUES ($1, $2, 'founder', $3, $4) RETURNING uuid", name, email, "hash", uuid).Scan(&outUUID)
	require.NoError(t, err)
	return outUUID
}

// CreateTestStartup inserts a startup for the given owner uuid and returns its ID.
func CreateTestStartup(t *testing.T, db *pgxpool.Pool, ownerUUID string) int {
	t.Helper()

	ctx := context.Background()
	suffix := nextSuffix()
	name := fmt.Sprintf("test-startup-%d", suffix)

	var id int64
	err := db.QueryRow(ctx, "INSERT INTO startups (name, owner_uuid, status) VALUES ($1, $2, 'active') RETURNING id", name, ownerUUID).Scan(&id)
	require.NoError(t, err)
	return int(id)
}

// CreateTestAsset inserts an active, unsold asset for the given user uuid and returns its ID.
func CreateTestAsset(t *testing.T, db *pgxpool.Pool, userUUID string) int {
	t.Helper()

	ctx := context.Background()
	suffix := nextSuffix()
	title := fmt.Sprintf("test-asset-%d", suffix)

	var id int64
	err := db.QueryRow(ctx, "INSERT INTO assets (user_uuid, title, asset_type) VALUES ($1, $2, 'research') RETURNING id", userUUID, title).Scan(&id)
	require.NoError(t, err)
	return int(id)
}
