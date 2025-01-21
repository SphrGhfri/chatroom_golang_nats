package websocket

import (
	"log"
	"strings"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
	gws "github.com/gorilla/websocket"
)

type Connection struct {
	Ws          *gws.Conn
	Send        chan interface{}
	Hub         *Hub
	Username    string
	ChatService service.ChatService
	Logger      logger.Logger
}

func (c *Connection) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		// Remove from presence if you handle that in the service:
		_ = c.ChatService.RemoveActiveUser(c.Username)
		c.Ws.Close()
	}()

	for {
		var msg domain.ChatMessage
		if err := c.Ws.ReadJSON(&msg); err != nil {
			c.Logger.Errorf("[CONNECTION] ReadJSON error: %v", err)
			break
		}

		switch msg.Type {

		case domain.MessageTypeList:
			// Example: list of active users
			users, err := c.ChatService.ListActiveUsers()
			if err != nil {
				c.Logger.Errorf("[CONNECTION] List users error: %v", err)
				continue
			}
			// Build a reply message for the requesting client
			reply := domain.ChatMessage{
				Type:    "list_users_response",
				Sender:  "System",
				Content: "Active users: " + strings.Join(users, ", "),
				// Possibly use msg.Timestamp or set a new timestamp
			}
			// Send only to this connection
			c.Send <- reply

		case domain.MessageTypeChat:
			// Normal chat message
			// Typically you'd set the sender, but maybe you're relying on the client to send it
			// If needed, set: msg.Sender = c.Username
			if err := c.ChatService.PublishMessage(msg); err != nil {
				c.Logger.Errorf("[CONNECTION] PublishMessage error: %v", err)
			}

		default:
			// Unknown or not handled
			c.Logger.Infof("[CONNECTION] Unknown message type: %s", msg.Type)
		}
	}
}

// writePump listens on the send channel and sends messages to the WebSocket.
func (c *Connection) WritePump() {
	defer c.Ws.Close()

	for msg := range c.Send {
		err := c.Ws.WriteJSON(msg)
		if err != nil {
			log.Printf("[CONNECTION] WriteJSON error: %v", err)
			break
		}
	}
}
