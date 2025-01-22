package ws

import (
	"fmt"
	"net/http"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	conn        *websocket.Conn
	username    string
	currentRoom string
	chatService service.ChatService
	logger      logger.Logger
}

func newClient(conn *websocket.Conn, username string, chatService service.ChatService, logger logger.Logger) *Client {
	return &Client{
		conn:        conn,
		username:    username,
		currentRoom: "global",
		chatService: chatService,
		logger:      logger,
	}
}

func (c *Client) initialize() error {
	if err := c.chatService.AddActiveUser(c.username); err != nil {
		return fmt.Errorf("failed to add active user: %w", err)
	}

	if err := c.chatService.JoinRoom("global", c.username, c.handleMessage); err != nil {
		c.chatService.RemoveActiveUser(c.username)
		return fmt.Errorf("failed to join global room: %w", err)
	}

	return nil
}

func checkUsernameExists(username string, chatService service.ChatService, logger logger.Logger, conn *websocket.Conn) error {
	exists, err := chatService.IsUserActive(username)
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

// Add new error handling function
func sendErrorMessageAndClose(conn *websocket.Conn, errMsg string) {
	errorMessage := domain.ChatMessage{
		Type:    domain.MessageTypeSystem,
		Content: errMsg,
	}
	conn.WriteJSON(errorMessage)
	conn.Close()
}

func HandleWebSocket(chatService service.ChatService, logger logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "username required", http.StatusBadRequest)
			return
		}

		// Upgrade connection first so we can send error messages through WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Errorf("websocket upgrade failed: %v", err)
			return
		}

		// Check if username exists
		if err := checkUsernameExists(username, chatService, logger, conn); err != nil {
			sendErrorMessageAndClose(conn, err.Error())
			return
		}

		client := newClient(conn, username, chatService, logger)
		if err := client.initialize(); err != nil {
			logger.Errorf("failed to initialize client: %v", err)
			sendErrorMessageAndClose(conn, "Failed to initialize connection")
			return
		}

		go client.readPump()
	}
}

func (c *Client) readPump() {
	defer func() {
		c.chatService.RemoveActiveUser(c.username)
		c.chatService.LeaveRoom(c.currentRoom, c.username)
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

func (c *Client) handleListCommand(msg domain.ChatMessage) {
	if msg.Room != "" {
		c.handleListRoomMembers(msg.Room)
	} else {
		c.handleListActiveUsers()
	}
}

func (c *Client) handleListRoomMembers(room string) {
	users, err := c.chatService.ListRoomMembers(room)
	if err != nil {
		c.logger.Errorf("failed to list room members: %v", err)
		return
	}
	c.sendSystemMessage(fmt.Sprintf("Users in room %s: %v", room, users))
}

func (c *Client) handleListActiveUsers() {
	users, err := c.chatService.ListActiveUsers()
	if err != nil {
		c.logger.Errorf("failed to list active users: %v", err)
		return
	}
	c.sendSystemMessage(fmt.Sprintf("Active users: %v", users))
}

func (c *Client) handleListRooms() {
	rooms, err := c.chatService.ListAllRooms()
	if err != nil {
		c.logger.Errorf("failed to list rooms: %v", err)
		return
	}
	c.sendSystemMessage(fmt.Sprintf("Available rooms: %v", rooms))
}

func (c *Client) handleJoinRoom(newRoom string) {
	if err := c.chatService.SwitchRoom(c.currentRoom, newRoom, c.username, c.handleMessage); err != nil {
		c.logger.Errorf("failed to switch room: %v", err)
		return
	}
	c.currentRoom = newRoom
}

func (c *Client) handleLeaveRoom() {
	if err := c.chatService.SwitchRoom(c.currentRoom, "global", c.username, c.handleMessage); err != nil {
		c.logger.Errorf("failed to return to global: %v", err)
		return
	}
	c.currentRoom = "global"
}

func (c *Client) handleChatMessage(msg domain.ChatMessage) {
	msg.Room = c.currentRoom
	if err := c.chatService.PublishMessage(msg); err != nil {
		c.logger.Errorf("failed to publish message: %v", err)
	}
}

func (c *Client) sendSystemMessage(content string) {
	c.handleMessage(domain.ChatMessage{
		Type:    domain.MessageTypeSystem,
		Content: content,
	})
}

func (c *Client) handleMessage(msg domain.ChatMessage) {
	if err := c.conn.WriteJSON(msg); err != nil {
		c.logger.Errorf("failed to write message to websocket: %v", err)
	}
}
