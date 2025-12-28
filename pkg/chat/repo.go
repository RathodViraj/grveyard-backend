package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageStore interface {
	SaveMessage(ctx context.Context, senderUUID, receiverUUID, content string, messageType int16, messagedAt int64) (int64, error)
	UpdateLastActive(ctx context.Context, userUUID string, lastActiveEpoch int64) error
	MarkMessagesAsRead(ctx context.Context, receiverUUID string, messageIDs []string) ([]string, error)
	GetConversationHistory(ctx context.Context, userUUID, peerUUID string, limit int, beforeEpoch int64) ([]MessageHistoryItem, error)
}

type PostgresMessageStore struct {
	pool *pgxpool.Pool
}

func NewPostgresMessageStore(pool *pgxpool.Pool) *PostgresMessageStore {
	return &PostgresMessageStore{pool: pool}
}

// SaveMessage inserts a message into the messages table using UUIDs to resolve user IDs.
// Returns the inserted DB message ID (bigint) or an error.
func (r *PostgresMessageStore) SaveMessage(ctx context.Context, senderUUID, receiverUUID, content string, messageType int16, messagedAt int64) (int64, error) {
	if r.pool == nil {
		return 0, errors.New("db pool is nil")
	}

	// Parameterized insert selecting ids from users by uuid
	const insertSQL = `
		INSERT INTO messages (sender_id, receiver_id, content, message_type, is_read, messaged_at)
		SELECT s.id, r.id, $3, $4, FALSE, $5
		FROM users s, users r
		WHERE s.uuid = $1 AND r.uuid = $2
		RETURNING id
	`

	var dbID int64
	// Use a context with reasonable timeout to avoid hung connections
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.pool.QueryRow(ctxTimeout, insertSQL, senderUUID, receiverUUID, content, messageType, messagedAt)
	if err := row.Scan(&dbID); err != nil {
		return 0, fmt.Errorf("insert message: %w", err)
	}
	return dbID, nil
}

// UpdateLastActive updates users.last_active_at with epoch seconds for the given user UUID.
func (r *PostgresMessageStore) UpdateLastActive(ctx context.Context, userUUID string, lastActiveEpoch int64) error {
	if r.pool == nil {
		return errors.New("db pool is nil")
	}

	const updateSQL = `
		UPDATE users
		SET last_active_at = $2
		WHERE uuid = $1
	`

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd, err := r.pool.Exec(ctxTimeout, updateSQL, userUUID, lastActiveEpoch)
	if err != nil {
		return fmt.Errorf("update last_active_at: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("no user found for uuid: %s", userUUID)
	}
	return nil
}

// MarkMessagesAsRead marks messages as read where receiver matches the given UUID.
// Returns the list of sender UUIDs who should be notified.
func (r *PostgresMessageStore) MarkMessagesAsRead(ctx context.Context, receiverUUID string, messageIDs []string) ([]string, error) {
	if r.pool == nil {
		return nil, errors.New("db pool is nil")
	}
	if len(messageIDs) == 0 {
		return nil, nil
	}

	// Convert string message IDs to int64 for DB query
	const updateSQL = `
		UPDATE messages m
		SET is_read = TRUE
		FROM users u
		WHERE m.receiver_id = u.id
		  AND u.uuid = $1
		  AND m.id = ANY($2)
		  AND m.is_read = FALSE
		RETURNING (SELECT s.uuid FROM users s WHERE s.id = m.sender_id) as sender_uuid
	`

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Convert message IDs from string to int64
	ids := make([]int64, 0, len(messageIDs))
	for _, idStr := range messageIDs {
		var id int64
		if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.pool.Query(ctxTimeout, updateSQL, receiverUUID, ids)
	if err != nil {
		return nil, fmt.Errorf("mark messages as read: %w", err)
	}
	defer rows.Close()

	senderUUIDs := make([]string, 0)
	seen := make(map[string]bool)
	for rows.Next() {
		var senderUUID string
		if err := rows.Scan(&senderUUID); err != nil {
			return nil, fmt.Errorf("scan sender uuid: %w", err)
		}
		if !seen[senderUUID] {
			senderUUIDs = append(senderUUIDs, senderUUID)
			seen[senderUUID] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return senderUUIDs, nil
}

// GetConversationHistory fetches message history between two users with pagination.
// Returns messages ordered by messaged_at ASC (oldest first).
func (r *PostgresMessageStore) GetConversationHistory(ctx context.Context, userUUID, peerUUID string, limit int, beforeEpoch int64) ([]MessageHistoryItem, error) {
	if r.pool == nil {
		return nil, errors.New("db pool is nil")
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100 // Cap at 100
	}

	const querySQL = `
		SELECT
			s.uuid as sender_uuid,
			r.uuid as receiver_uuid,
			m.content,
			m.message_type,
			m.is_read,
			m.messaged_at
		FROM messages m
		JOIN users s ON m.sender_id = s.id
		JOIN users r ON m.receiver_id = r.id
		WHERE (
			(s.uuid = $1 AND r.uuid = $2)
			OR
			(s.uuid = $2 AND r.uuid = $1)
		)
		AND m.messaged_at < $3
		ORDER BY m.messaged_at ASC
		LIMIT $4
	`

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.pool.Query(ctxTimeout, querySQL, userUUID, peerUUID, beforeEpoch, limit)
	if err != nil {
		return nil, fmt.Errorf("query conversation history: %w", err)
	}
	defer rows.Close()

	result := make([]MessageHistoryItem, 0, limit)
	for rows.Next() {
		var item MessageHistoryItem
		if err := rows.Scan(&item.SenderID, &item.ReceiverID, &item.Content, &item.MessageType, &item.IsRead, &item.MessagedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	return result, nil
}
