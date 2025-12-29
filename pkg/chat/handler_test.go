package chat

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// mockStore is a lightweight MessageStore double for unit testing handler logic.
type mockStore struct {
	saveCalls []struct {
		sender   string
		receiver string
		content  string
		typeID   int16
		ts       int64
	}
	saveErr       error
	markErr       error
	updateErr     error
	historyResult []MessageHistoryItem
}

func (m *mockStore) SaveMessage(ctx context.Context, senderUUID, receiverUUID, content string, messageType int16, messagedAt int64) (int64, error) {
	m.saveCalls = append(m.saveCalls, struct {
		sender   string
		receiver string
		content  string
		typeID   int16
		ts       int64
	}{senderUUID, receiverUUID, content, messageType, messagedAt})
	if m.saveErr != nil {
		return 0, m.saveErr
	}
	return 1, nil
}

func (m *mockStore) UpdateLastActive(ctx context.Context, userUUID string, lastActiveEpoch int64) error {
	return m.updateErr
}

func (m *mockStore) MarkMessagesAsRead(ctx context.Context, receiverUUID string, messageIDs []string) ([]string, error) {
	if m.markErr != nil {
		return nil, m.markErr
	}
	return []string{"sender-online"}, nil
}

func (m *mockStore) GetConversationHistory(ctx context.Context, userUUID, peerUUID string, limit int, beforeEpoch int64) ([]MessageHistoryItem, error) {
	return m.historyResult, nil
}

// TestValidateMessage covers payload validation rules without websockets.
func TestValidateMessage(t *testing.T) {
	handler := NewHandler(NewConnectionManager())

	tests := []struct {
		name    string
		msg     Message
		sender  string
		wantErr bool
	}{
		{"empty content", Message{ReceiverID: "user2", Content: ""}, "user1", true},
		{"self message", Message{ReceiverID: "user1", Content: "hi"}, "user1", true},
		{"missing receiver", Message{ReceiverID: "", Content: "hi"}, "user1", true},
		{"valid message", Message{ReceiverID: "user2", Content: "hi"}, "user1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateMessage(tt.msg, tt.sender)
			require.Equal(t, tt.wantErr, err != nil)
		})
	}
}

// TestProcessMessage_OfflineAck ensures queued ack and no panic when receiver is offline.
func TestProcessMessage_OfflineAck(t *testing.T) {
	manager := NewConnectionManager()
	store := &mockStore{}
	handler := NewHandler(manager)
	handler.SetRepository(store)

	client := &Client{UserID: "user1", Send: make(chan interface{}, 1), Done: make(chan struct{})}
	msg := Message{ReceiverID: "offline", Content: "hi"}

	handler.processMessage(client, msg)

	select {
	case raw := <-client.Send:
		ack, ok := raw.(Acknowledgement)
		require.True(t, ok)
		require.Equal(t, "queued", ack.Status)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for ack")
	}
	require.Len(t, store.saveCalls, 1)
}

// TestProcessMessage_OnlineDelivered ensures message forwarded and ack marked sent.
func TestProcessMessage_OnlineDelivered(t *testing.T) {
	manager := NewConnectionManager()
	receiver := manager.AddClient("user2", nil)
	receiver.Send = make(chan interface{}, 1)
	store := &mockStore{}
	handler := NewHandler(manager)
	handler.SetRepository(store)

	client := &Client{UserID: "user1", Send: make(chan interface{}, 1), Done: make(chan struct{})}
	msg := Message{ReceiverID: "user2", Content: "hi"}

	handler.processMessage(client, msg)

	// Sender ack
	select {
	case raw := <-client.Send:
		ack := raw.(Acknowledgement)
		require.Equal(t, "sent", ack.Status)
	case <-time.After(1 * time.Second):
		t.Fatal("no ack")
	}

	// Receiver message
	select {
	case raw := <-receiver.Send:
		recv := raw.(Message)
		require.Equal(t, "hi", recv.Content)
		require.Equal(t, "user1", recv.SenderID)
	case <-time.After(1 * time.Second):
		t.Fatal("no forwarded message")
	}

	require.Len(t, store.saveCalls, 1)
}

// TestProcessMessage_SaveError returns error ack and no forward.
func TestProcessMessage_SaveError(t *testing.T) {
	manager := NewConnectionManager()
	receiver := manager.AddClient("user2", nil)
	receiver.Send = make(chan interface{}, 1)
	store := &mockStore{saveErr: errors.New("db down")}
	handler := NewHandler(manager)
	handler.SetRepository(store)

	client := &Client{UserID: "user1", Send: make(chan interface{}, 1), Done: make(chan struct{})}
	msg := Message{ReceiverID: "user2", Content: "hi"}

	handler.processMessage(client, msg)

	select {
	case raw := <-client.Send:
		ack := raw.(Acknowledgement)
		require.Equal(t, "error", ack.Status)
	case <-time.After(1 * time.Second):
		t.Fatal("no error ack")
	}

	select {
	case <-receiver.Send:
		t.Fatal("should not forward on save error")
	default:
	}
}

// TestProcessMessage_SelfMessageRejected ensures validation stops persistence.
func TestProcessMessage_SelfMessageRejected(t *testing.T) {
	manager := NewConnectionManager()
	store := &mockStore{}
	handler := NewHandler(manager)
	handler.SetRepository(store)

	client := &Client{UserID: "user1", Send: make(chan interface{}, 1), Done: make(chan struct{})}
	msg := Message{ReceiverID: "user1", Content: "self"}

	handler.processMessage(client, msg)

	select {
	case raw := <-client.Send:
		_, ok := raw.(ErrorResponse)
		require.True(t, ok)
	case <-time.After(1 * time.Second):
		t.Fatal("no error response")
	}
	require.Empty(t, store.saveCalls)
}

// mockUpgrader allows testing that the handler uses the injected upgrader.
type mockUpgrader struct{ called bool }

func (m *mockUpgrader) Upgrade(w http.ResponseWriter, r *http.Request, _ http.Header) (*websocket.Conn, error) {
	m.called = true
	return nil, errors.New("upgrade failed (test)")
}

// TestHandleWebSocket_UsesInjectedUpgrader verifies the handler calls the configured upgrader
// and handles upgrade failure without adding a client.
func TestHandleWebSocket_UsesInjectedUpgrader(t *testing.T) {
	manager := NewConnectionManager()
	handler := NewHandler(manager)

	mu := &mockUpgrader{}
	handler.SetWebSocketUpgrader(mu)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws/chat?user_id=userX", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", "userX"))

	handler.HandleWebSocket(rr, req)

	require.True(t, mu.called, "expected upgrader to be called")
	require.False(t, manager.IsOnline("userX"), "user should not be online after failed upgrade")
}
