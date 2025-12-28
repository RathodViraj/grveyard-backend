package chat

import (
	"time"
)

// Message represents a chat message
type Message struct {
	SenderID    string    `json:"sender_id"`
	ReceiverID  string    `json:"receiver_id"`
	Content     string    `json:"content"`
	Timestamp   time.Time `json:"timestamp"`
	ID          string    `json:"id"` // Unique message ID
	MessageType int16     `json:"message_type,omitempty"`
	IsRead      bool      `json:"is_read,omitempty"`
}

// Acknowledgement sent to sender when message is processed
type Acknowledgement struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"` // "sent" or "delivered"
	Error     string `json:"error,omitempty"`
}

// ErrorResponse sent to client on errors
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// ReadReceipt sent by receiver to mark messages as read
type ReadReceipt struct {
	EventType  string   `json:"event_type"`  // "message_read"
	MessageIDs []string `json:"message_ids"` // UUIDs of messages to mark as read
}

// ReadReceiptNotification sent to sender when their messages are read
type ReadReceiptNotification struct {
	EventType  string   `json:"event_type"` // "message_read"
	MessageIDs []string `json:"message_ids"`
	ReadBy     string   `json:"read_by"` // UUID of user who read the messages
}

// MessageHistoryItem represents a message in conversation history (REST API)
type MessageHistoryItem struct {
	SenderID    string `json:"sender_id"`   // UUID
	ReceiverID  string `json:"receiver_id"` // UUID
	Content     string `json:"content"`
	MessageType int16  `json:"message_type"`
	IsRead      bool   `json:"is_read"`
	MessagedAt  int64  `json:"messaged_at"` // epoch seconds
}
