package chat

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a connected user
type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan interface{} // Channel to send messages to this client
	Done   chan struct{}    // Signal to stop reading/writing
}

// ConnectionManager manages all active WebSocket connections
type ConnectionManager struct {
	mu      sync.RWMutex
	clients map[string]*Client // user_id -> Client
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		clients: make(map[string]*Client),
	}
}

// AddClient registers a new client connection
func (cm *ConnectionManager) AddClient(userID string, conn *websocket.Conn) *Client {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Disconnect existing connection for this user if any
	if existing, ok := cm.clients[userID]; ok {
		close(existing.Done)
		existing.Conn.Close()
	}

	client := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan interface{}, 32), // Buffered channel to handle bursts
		Done:   make(chan struct{}),
	}

	cm.clients[userID] = client
	return client
}

// RemoveClient unregisters a client connection
func (cm *ConnectionManager) RemoveClient(userID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if client, ok := cm.clients[userID]; ok {
		close(client.Done)
		delete(cm.clients, userID)
	}
}

// GetClient retrieves a client by user ID
func (cm *ConnectionManager) GetClient(userID string) *Client {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.clients[userID]
}

// IsOnline checks if a user is currently online
func (cm *ConnectionManager) IsOnline(userID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	_, exists := cm.clients[userID]
	return exists
}

// GetOnlineUsers returns a list of all online user IDs
func (cm *ConnectionManager) GetOnlineUsers() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	users := make([]string, 0, len(cm.clients))
	for userID := range cm.clients {
		users = append(users, userID)
	}
	return users
}

// BroadcastToUser sends a message to a specific user
// Returns error if user is not online
func (cm *ConnectionManager) BroadcastToUser(userID string, message interface{}) error {
	cm.mu.RLock()
	client, ok := cm.clients[userID]
	cm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("user %s is not online", userID)
	}

	select {
	case client.Send <- message:
		return nil
	case <-client.Done:
		// Client disconnected while we were sending
		return fmt.Errorf("user %s disconnected", userID)
	default:
		// Channel full - should not happen with buffered channel, but handle gracefully
		return fmt.Errorf("user %s message queue full", userID)
	}
}
