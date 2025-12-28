package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"grveyard/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Handler wraps the connection manager and provides HTTP handlers
type Handler struct {
	manager *ConnectionManager
	// Optional: logger can be injected
	logger interface {
		Printf(string, ...interface{})
	}
	repo MessageStore // optional; if nil, persistence is skipped
}

// NewHandler creates a new chat handler
func NewHandler(manager *ConnectionManager) *Handler {
	return &Handler{
		manager: manager,
		logger:  log.New(log.Writer(), "[chat] ", log.LstdFlags),
	}
}

// SetRepository injects the message store for persistence (kept name for compatibility)
func (h *Handler) SetRepository(r MessageStore) {
	h.repo = r
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, validate origin properly
		return true
	},
}

// HandleWebSocket handles the WebSocket upgrade and connection
// Expects user_id to be set in the request context during authentication middleware
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get user_id from context (set by authentication middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		http.Error(w, "unauthorized: user_id not found", http.StatusUnauthorized)
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Printf("websocket upgrade error: %v", err)
		return
	}

	// Add client to manager
	client := h.manager.AddClient(userID, conn)
	h.logger.Printf("user %s connected", userID)

	// Update last_active_at on connect (epoch seconds)
	if h.repo != nil {
		if err := h.repo.UpdateLastActive(context.Background(), userID, time.Now().Unix()); err != nil {
			h.logger.Printf("last_active_at update (connect) failed for %s: %v", userID, err)
		}
	}

	// Start goroutines for reading and writing
	go h.readLoop(client)
	go h.writeLoop(client)
}

// HandleWebSocketGin validates user_id from query, injects into context, and upgrades to WebSocket.
func (h *Handler) HandleWebSocketGin(c *gin.Context) {
	uid := c.Query("user_id")
	if _, err := uuid.Parse(uid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id, must be UUID"})
		return
	}

	ctx := context.WithValue(c.Request.Context(), "user_id", uid)
	req := c.Request.WithContext(ctx)
	h.HandleWebSocket(c.Writer, req)
}

// readLoop reads messages from the WebSocket connection
func (h *Handler) readLoop(client *Client) {
	defer func() {
		h.manager.RemoveClient(client.UserID)
		client.Conn.Close()
		h.logger.Printf("user %s disconnected", client.UserID)

		// Update last_active_at on disconnect (epoch seconds)
		if h.repo != nil {
			if err := h.repo.UpdateLastActive(context.Background(), client.UserID, time.Now().Unix()); err != nil {
				h.logger.Printf("last_active_at update (disconnect) failed for %s: %v", client.UserID, err)
			}
		}
	}()

	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-client.Done:
			return
		default:
		}

		var rawMsg map[string]interface{}
		err := client.Conn.ReadJSON(&rawMsg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Printf("websocket error for user %s: %v", client.UserID, err)
			}
			return
		}

		// Check event_type to determine message or read receipt
		if eventType, ok := rawMsg["event_type"].(string); ok && eventType == "message_read" {
			// Handle read receipt
			go h.processReadReceipt(client, rawMsg)
		} else {
			// Handle regular message
			var msg Message
			// Re-unmarshal into Message struct
			msgBytes, _ := json.Marshal(rawMsg)
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				h.sendError(client, Message{}, "invalid message format")
				continue
			}
			go h.processMessage(client, msg)
		}
	}
}

// IsUserOnline reports if a given user has an active WS connection
func (h *Handler) IsUserOnline(userID string) bool {
	return h.manager.IsOnline(userID)
}

// writeLoop writes messages to the WebSocket connection
func (h *Handler) writeLoop(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-client.Done:
			return

		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if !ok {
				// Channel closed
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := client.Conn.WriteJSON(message)
			if err != nil {
				h.logger.Printf("write error for user %s: %v", client.UserID, err)
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.logger.Printf("ping error for user %s: %v", client.UserID, err)
				return
			}
		}
	}
}

// processMessage validates and handles incoming messages
func (h *Handler) processMessage(client *Client, msg Message) {
	// Validate message structure
	if err := h.validateMessage(msg, client.UserID); err != nil {
		h.sendError(client, msg, err.Error())
		return
	}

	// Generate message ID if not provided
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	}

	// Ensure sender_id matches authenticated user
	msg.SenderID = client.UserID

	// Default message type if not provided
	if msg.MessageType == 0 {
		msg.MessageType = 0 // text
	}

	// Persist synchronously after validation and before forwarding
	if h.repo != nil {
		epoch := msg.Timestamp.Unix()
		if _, err := h.repo.SaveMessage(context.Background(), msg.SenderID, msg.ReceiverID, msg.Content, msg.MessageType, epoch); err != nil {
			// Log and send error acknowledgement without crashing
			h.logger.Printf("db insert failed for user %s -> %s: %v", msg.SenderID, msg.ReceiverID, err)
			ack := Acknowledgement{MessageID: msg.ID, Status: "error", Error: "failed to persist message"}
			select {
			case client.Send <- ack:
			case <-client.Done:
			}
			return
		}
	}

	// Check if receiver is online
	if h.manager.IsOnline(msg.ReceiverID) {
		// Forward message to receiver
		err := h.manager.BroadcastToUser(msg.ReceiverID, msg)
		if err != nil {
			h.sendError(client, msg, fmt.Sprintf("failed to deliver message: %v", err))
			return
		}
	}

	// Acknowledge to sender immediately
	ack := Acknowledgement{
		MessageID: msg.ID,
		Status:    "sent",
	}
	if !h.manager.IsOnline(msg.ReceiverID) {
		ack.Status = "queued" // Receiver offline but message was recorded
	}

	select {
	case client.Send <- ack:
	case <-client.Done:
		// Client disconnected
	}
}

// validateMessage validates the message before processing
func (h *Handler) validateMessage(msg Message, senderID string) error {
	if msg.Content == "" {
		return fmt.Errorf("message content cannot be empty")
	}

	if len(msg.Content) > 10000 {
		return fmt.Errorf("message content too long (max 10000 characters)")
	}

	if msg.ReceiverID == "" {
		return fmt.Errorf("receiver_id is required")
	}

	// Reject self-messages
	if msg.ReceiverID == senderID {
		return fmt.Errorf("cannot send messages to yourself")
	}

	return nil
}

// sendError sends an error response to the client
func (h *Handler) sendError(client *Client, originalMsg Message, errMsg string) {
	errResp := ErrorResponse{
		Error: errMsg,
	}

	select {
	case client.Send <- errResp:
	case <-client.Done:
		// Client disconnected
	}
}

// processReadReceipt handles read receipt events from the receiver
func (h *Handler) processReadReceipt(client *Client, rawMsg map[string]interface{}) {
	if h.repo == nil {
		return // No DB support, skip
	}

	// Extract message IDs
	messageIDsRaw, ok := rawMsg["message_ids"].([]interface{})
	if !ok || len(messageIDsRaw) == 0 {
		h.sendError(client, Message{}, "message_ids required for read receipt")
		return
	}

	messageIDs := make([]string, 0, len(messageIDsRaw))
	for _, idRaw := range messageIDsRaw {
		if idStr, ok := idRaw.(string); ok {
			messageIDs = append(messageIDs, idStr)
		}
	}

	if len(messageIDs) == 0 {
		return
	}

	// Mark messages as read in DB (only where receiver_id = client.UserID)
	senderUUIDs, err := h.repo.MarkMessagesAsRead(context.Background(), client.UserID, messageIDs)
	if err != nil {
		h.logger.Printf("failed to mark messages as read for %s: %v", client.UserID, err)
		h.sendError(client, Message{}, "failed to mark messages as read")
		return
	}

	// Notify senders if they are online
	notification := ReadReceiptNotification{
		EventType:  "message_read",
		MessageIDs: messageIDs,
		ReadBy:     client.UserID,
	}

	for _, senderUUID := range senderUUIDs {
		if h.manager.IsOnline(senderUUID) {
			if err := h.manager.BroadcastToUser(senderUUID, notification); err != nil {
				h.logger.Printf("failed to send read receipt to %s: %v", senderUUID, err)
			}
		}
	}
}

// Gin-specific wrappers using SendAPIResponse
// GetStatusGin godoc
// @Summary Get online users
// @Description Returns list of currently connected users
// @Tags chat
// @Produce json
// @Success 200 {object} response.APIResponse
// @Router /chat/status [get]
func (h *Handler) GetStatusGin(c *gin.Context) {
	users := h.manager.GetOnlineUsers()
	response.SendAPIResponse(c, http.StatusOK, true, "online status", map[string]interface{}{
		"online_users": users,
		"count":        len(users),
	})
}

// GetMessagesGin godoc
// @Summary Get conversation history
// @Description Fetch chat messages between the requesting user and a peer
// @Tags chat
// @Param user_id query string true "Requesting user UUID"
// @Param peer_id query string true "Peer user UUID"
// @Param limit query int false "Maximum messages to return (max 100)"
// @Param before query int false "Epoch seconds cursor for pagination"
// @Produce json
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Failure 403 {object} response.APIResponse
// @Failure 500 {object} response.APIResponse
// @Router /messages [get]
func (h *Handler) GetMessagesGin(c *gin.Context) {
	if h.repo == nil {
		response.SendAPIResponse(c, http.StatusServiceUnavailable, false, "message history not available", nil)
		return
	}

	uid := c.Query("user_id")
	if _, err := uuid.Parse(uid); err != nil {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid user_id, must be UUID", nil)
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", uid)
	c.Request = c.Request.WithContext(ctx)

	userID := uid
	queryUserID := c.Query("user_id")
	peerID := c.Query("peer_id")

	if queryUserID != userID {
		response.SendAPIResponse(c, http.StatusForbidden, false, "forbidden: can only fetch your own messages", nil)
		return
	}
	if peerID == "" {
		response.SendAPIResponse(c, http.StatusBadRequest, false, "peer_id is required", nil)
		return
	}

	// Parse limit and before
	limit := 50
	if ls := c.Query("limit"); ls != "" {
		if _, err := fmt.Sscanf(ls, "%d", &limit); err != nil {
			response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid limit parameter", nil)
			return
		}
	}
	beforeEpoch := time.Now().Unix()
	if bs := c.Query("before"); bs != "" {
		if _, err := fmt.Sscanf(bs, "%d", &beforeEpoch); err != nil {
			response.SendAPIResponse(c, http.StatusBadRequest, false, "invalid before parameter", nil)
			return
		}
	}

	messages, err := h.repo.GetConversationHistory(c.Request.Context(), userID, peerID, limit, beforeEpoch)
	if err != nil {
		h.logger.Printf("failed to fetch messages for %s <-> %s: %v", userID, peerID, err)
		response.SendAPIResponse(c, http.StatusInternalServerError, false, "failed to fetch messages", nil)
		return
	}

	response.SendAPIResponse(c, http.StatusOK, true, "messages", map[string]interface{}{
		"messages": messages,
		"count":    len(messages),
	})
}

// AuthMiddleware removed; Gin routes should handle auth and context injection
