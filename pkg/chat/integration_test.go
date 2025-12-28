package chat

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"

	"grveyard/pkg/testhelpers"
)

// newTestPool connects to a real Postgres instance for integration tests.
// Skips if DATABASE_URL_FOR_TEST is not set to keep CI deterministic.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if err := godotenv.Load(); err != nil {
		t.Log("No .env file found, using environment variables")
	}
	dsn := os.Getenv("DATABASE_URL_FOR_TEST")
	if dsn == "" {
		t.Skip("DATABASE_URL_FOR_TEST not set; skipping integration tests")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.MaxConns = 4

	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)

	// truncate := func() {
	// 	_, err := pool.Exec(context.Background(), "TRUNCATE messages, users RESTART IDENTITY CASCADE")
	// 	require.NoError(t, err)
	// }
	// truncate()
	// t.Cleanup(truncate)
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestSaveMessage_PersistsFields(t *testing.T) {
	pool := newTestPool(t)
	store := NewPostgresMessageStore(pool)

	sender := testhelpers.CreateTestUser(t, pool)
	receiver := testhelpers.CreateTestUser(t, pool)
	messagedAt := time.Now().Unix()

	_, err := store.SaveMessage(context.Background(), sender, receiver, "hello", 1, messagedAt)
	require.NoError(t, err)

	row := pool.QueryRow(context.Background(), `
		SELECT s.uuid, r.uuid, m.content, m.message_type, m.is_read, m.messaged_at
		FROM messages m
		JOIN users s ON m.sender_id = s.id
		JOIN users r ON m.receiver_id = r.id
	`)
	var sUUID, rUUID, content string
	var messageType int16
	var isRead bool
	var storedAt int64
	require.NoError(t, row.Scan(&sUUID, &rUUID, &content, &messageType, &isRead, &storedAt))
	require.Equal(t, sender, sUUID)
	require.Equal(t, receiver, rUUID)
	require.Equal(t, "hello", content)
	require.Equal(t, int16(1), messageType)
	require.False(t, isRead)
	require.Equal(t, messagedAt, storedAt)
}

func TestProcessMessage_SelfMessageDoesNotPersist(t *testing.T) {
	pool := newTestPool(t)
	store := NewPostgresMessageStore(pool)
	manager := NewConnectionManager()
	handler := NewHandler(manager)
	handler.SetRepository(store)

	user := testhelpers.CreateTestUser(t, pool)
	client := &Client{UserID: user, Send: make(chan interface{}, 1), Done: make(chan struct{})}
	msg := Message{ReceiverID: user, Content: "self"}

	handler.processMessage(client, msg)

	// No insert expected
	var count int
	err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM messages").Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)

	select {
	case raw := <-client.Send:
		_, ok := raw.(ErrorResponse)
		require.True(t, ok)
	case <-time.After(1 * time.Second):
		t.Fatal("expected error response")
	}
}

func TestConversationHistory_BidirectionalAndOrdering(t *testing.T) {
	pool := newTestPool(t)
	store := NewPostgresMessageStore(pool)

	a := testhelpers.CreateTestUser(t, pool)
	b := testhelpers.CreateTestUser(t, pool)

	// Interleave messages A->B and B->A
	_, err := store.SaveMessage(context.Background(), a, b, "m1", 0, 100)
	require.NoError(t, err)
	_, err = store.SaveMessage(context.Background(), b, a, "m2", 0, 200)
	require.NoError(t, err)
	_, err = store.SaveMessage(context.Background(), a, b, "m3", 0, 300)
	require.NoError(t, err)

	messages, err := store.GetConversationHistory(context.Background(), a, b, 10, time.Now().Unix())
	require.NoError(t, err)
	require.Len(t, messages, 3)
	require.Equal(t, []string{"m1", "m2", "m3"}, []string{messages[0].Content, messages[1].Content, messages[2].Content})
	require.True(t, messages[0].MessagedAt < messages[1].MessagedAt && messages[1].MessagedAt < messages[2].MessagedAt)
}

func TestConversationHistory_PaginationBefore(t *testing.T) {
	pool := newTestPool(t)
	store := NewPostgresMessageStore(pool)

	a := testhelpers.CreateTestUser(t, pool)
	b := testhelpers.CreateTestUser(t, pool)

	store.SaveMessage(context.Background(), a, b, "old", 0, 100)
	store.SaveMessage(context.Background(), a, b, "mid", 0, 200)
	store.SaveMessage(context.Background(), a, b, "new", 0, 300)

	messages, err := store.GetConversationHistory(context.Background(), a, b, 10, 250)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, []string{"old", "mid"}, []string{messages[0].Content, messages[1].Content})
}

func TestMarkMessagesAsRead_OnlyReceiverCanAcknowledge(t *testing.T) {
	pool := newTestPool(t)
	store := NewPostgresMessageStore(pool)

	sender := testhelpers.CreateTestUser(t, pool)
	receiver := testhelpers.CreateTestUser(t, pool)
	other := testhelpers.CreateTestUser(t, pool)

	_, err := store.SaveMessage(context.Background(), sender, receiver, "hello", 0, 123)
	require.NoError(t, err)
	_, err = store.SaveMessage(context.Background(), sender, receiver, "hello2", 0, 124)
	require.NoError(t, err)

	// Attempt with non-receiver should not mark as read
	updated, err := store.MarkMessagesAsRead(context.Background(), other, []string{"1", "2"})
	require.NoError(t, err)
	require.Empty(t, updated)

	// Receiver marks as read
	updated, err = store.MarkMessagesAsRead(context.Background(), receiver, []string{"1", "2"})
	require.NoError(t, err)
	require.NotEmpty(t, updated)

	var unread int
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM messages WHERE is_read=false").Scan(&unread))
	require.Zero(t, unread)
}

func TestUpdateLastActive_Monotonic(t *testing.T) {
	pool := newTestPool(t)
	store := NewPostgresMessageStore(pool)
	user := testhelpers.CreateTestUser(t, pool)

	require.NoError(t, store.UpdateLastActive(context.Background(), user, 100))
	require.NoError(t, store.UpdateLastActive(context.Background(), user, 200))

	var lastActive int64
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT last_active_at FROM users WHERE uuid=$1", user).Scan(&lastActive))
	require.Equal(t, int64(200), lastActive)
}
