package ws

import (
	"context"
	"fmt"
	"net/http"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
	"github.com/gorilla/websocket"
)

// Core WebSocket configuration
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client represents a connected WebSocket client
type Client struct {
	conn        *websocket.Conn
	ctx         context.Context
	cancel      context.CancelFunc
	username    string
	currentRoom string
	chatService service.ChatService
	logger      logger.Logger
}

// === Core WebSocket Handler Functions ===

// HandleWebSocket is the main WebSocket connection handler
func HandleWebSocket(chatService service.ChatService, rootCtx context.Context, log logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create client-specific context
		clientCtx, clientCancel := context.WithCancel(rootCtx)

		username := r.URL.Query().Get("username")
		clientLog := log.WithFields(map[string]interface{}{
			"username":    username,
			"remote_addr": r.RemoteAddr,
		})

		if username == "" {
			clientLog.Warnf("Missing username in connection request")
			http.Error(w, "username required", http.StatusBadRequest)
			clientCancel()
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			clientLog.Errorf("WebSocket upgrade failed: %v", err)
			clientCancel()
			return
		}

		// Check username with client context
		if err := checkUsernameExists(clientCtx, username, chatService, clientLog, conn); err != nil {
			sendErrorMessageAndClose(conn, err.Error())
			clientCancel()
			return
		}

		client := newClient(clientCtx, clientCancel, conn, username, chatService, clientLog)
		if err := client.initialize(); err != nil {
			clientLog.Errorf("Failed to initialize client: %v", err)
			sendErrorMessageAndClose(conn, "Failed to initialize connection")
			clientCancel()
			return
		}

		go client.readPump()
	}
}

// readPump handles incoming WebSocket messages
func (c *Client) readPump() {
	defer func() {
		c.cancel() // Cancel client context
		c.chatService.RemoveActiveUser(c.ctx, c.username)
		c.chatService.LeaveRoom(c.ctx, c.currentRoom, c.username)
		c.conn.Close()
	}()

	for {
		var msg domain.ChatMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Errorf("read error: %v", err)
			}
			break
		}

		msg.Sender = c.username

		switch msg.Type {
		case domain.MessageTypeList:
			c.handleListCommand(msg)
		case domain.MessageTypeRooms:
			c.handleListRooms()
		case domain.MessageTypeJoin:
			c.handleJoinRoom(msg.Room)
		case domain.MessageTypeLeave:
			c.handleLeaveRoom()
		case domain.MessageTypeChat:
			c.handleChatMessage(msg)
		}
	}
}

// === Client Lifecycle Management ===

// newClient creates a new WebSocket client instance
func newClient(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, username string, chatService service.ChatService, log logger.Logger) *Client {
	return &Client{
		conn:        conn,
		ctx:         ctx,
		cancel:      cancel,
		username:    username,
		currentRoom: "global",
		chatService: chatService,
		logger:      log,
	}
}

// initialize sets up the client's initial state
func (c *Client) initialize() error {
	if err := c.chatService.AddActiveUser(c.ctx, c.username); err != nil {
		return fmt.Errorf("failed to add active user: %w", err)
	}

	if err := c.chatService.JoinRoom(c.ctx, "global", c.username, c.handleMessage); err != nil {
		c.chatService.RemoveActiveUser(c.ctx, c.username)
		return fmt.Errorf("failed to join global room: %w", err)
	}

	return nil
}

// checkUsernameExists verifies if the username is already in use
func checkUsernameExists(ctx context.Context, username string, chatService service.ChatService, logger logger.Logger, conn *websocket.Conn) error {
	exists, err := chatService.IsUserActive(ctx, username)
	if err != nil {
		logger.Errorf("failed to check username existence: %v", err)
		return fmt.Errorf("internal server error")
	}

	if exists {
		logger.Infof("username %s is already taken", username)
		sendErrorMessageAndClose(conn, "username already exists")
		return fmt.Errorf("username already exists")
	}

	return nil
}

// === Message Handling Functions ===

// handleMessage sends a message to the WebSocket client
func (c *Client) handleMessage(msg domain.ChatMessage) {
	if err := c.conn.WriteJSON(msg); err != nil {
		c.logger.Errorf("failed to write message to websocket: %v", err)
	}
}

// handleChatMessage processes and publishes chat messages
func (c *Client) handleChatMessage(msg domain.ChatMessage) {
	msg.Room = c.currentRoom
	if err := c.chatService.PublishMessage(c.ctx, msg); err != nil {
		c.logger.Errorf("failed to publish message: %v", err)
	}
}

// sendSystemMessage sends system notifications to the client
func (c *Client) sendSystemMessage(content string) {
	c.handleMessage(domain.ChatMessage{
		Type:    domain.MessageTypeSystem,
		Content: content,
	})
}

// sendErrorMessageAndClose sends an error message and closes the connection
func sendErrorMessageAndClose(conn *websocket.Conn, errMsg string) {
	errorMessage := domain.ChatMessage{
		Type:    domain.MessageTypeSystem,
		Content: errMsg,
	}
	conn.WriteJSON(errorMessage)
	conn.Close()
}

// === Room Management Functions ===

// handleJoinRoom processes room join requests
func (c *Client) handleJoinRoom(newRoom string) {
	if err := c.chatService.SwitchRoom(c.ctx, c.currentRoom, newRoom, c.username, c.handleMessage); err != nil {
		c.logger.Errorf("failed to switch room: %v", err)
		return
	}
	c.currentRoom = newRoom
}

// handleLeaveRoom processes room leave requests
func (c *Client) handleLeaveRoom() {
	if err := c.chatService.SwitchRoom(c.ctx, c.currentRoom, "global", c.username, c.handleMessage); err != nil {
		c.logger.Errorf("failed to return to global: %v", err)
		return
	}
	c.currentRoom = "global"
}

// === List/Query Functions ===

// handleListCommand processes list commands for users
func (c *Client) handleListCommand(msg domain.ChatMessage) {
	if msg.Room != "" {
		c.handleListRoomMembers(msg.Room)
	} else {
		c.handleListActiveUsers()
	}
}

// handleListRoomMembers retrieves and sends room member list
func (c *Client) handleListRoomMembers(room string) {
	users, err := c.chatService.ListRoomMembers(c.ctx, room)
	if err != nil {
		c.logger.Errorf("failed to list room members: %v", err)
		return
	}
	c.sendSystemMessage(fmt.Sprintf("Users in room %s: %v", room, users))
}

// handleListActiveUsers retrieves and sends active users list
func (c *Client) handleListActiveUsers() {
	users, err := c.chatService.ListActiveUsers(c.ctx)
	if err != nil {
		c.logger.Errorf("failed to list active users: %v", err)
		return
	}
	c.sendSystemMessage(fmt.Sprintf("Active users: %v", users))
}

// handleListRooms retrieves and sends available rooms list
func (c *Client) handleListRooms() {
	rooms, err := c.chatService.ListAllRooms(c.ctx)
	if err != nil {
		c.logger.Errorf("failed to list rooms: %v", err)
		return
	}
	c.sendSystemMessage(fmt.Sprintf("Available rooms: %v", rooms))
}
