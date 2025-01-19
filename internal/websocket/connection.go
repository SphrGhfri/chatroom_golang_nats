package websocket

import (
	"log"
	"net/http"
	"strings"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for testing; restrict in production.
	},
}

// Connection represents a single WebSocket connection to a client.
type Connection struct {
	ws          *websocket.Conn         // WebSocket connection
	send        chan domain.ChatMessage // Channel for sending messages to the client
	hub         *Hub                    // Reference to the Hub managing this connection
	redisClient *redis.RedisClient
	username    string
}

// ServeWS upgrades an HTTP connection to a WebSocket connection and registers it with the Hub.
func ServeWS(hub *Hub, redisClient *redis.RedisClient, w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[CONNECTION] Failed to upgrade WebSocket: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusInternalServerError)
		return
	}

	// Retrieve username from query parameter
	username := r.URL.Query().Get("username")
	if username == "" {
		log.Printf("Username is required")
		conn.Close()
		return
	}

	client := &Connection{
		ws:          conn,
		send:        make(chan domain.ChatMessage, 256), // Buffered channel for outgoing messages
		hub:         hub,
		redisClient: redisClient,
		username:    username,
	}

	// Add user to active users list
	err = redisClient.AddActiveUser(username)
	if err != nil {
		log.Printf("Failed to add user to Redis: %v", err)
	}
	// Register the client with the Hub
	hub.register <- client
	log.Printf("[CONNECTION] New WebSocket connection: %s Username: %s", conn.RemoteAddr(), username)

	go client.readPump()
	go client.writePump()
}

func (c *Connection) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.ws.Close()
		c.redisClient.RemoveActiveUser(c.username)
	}()

	for {
		var msg domain.ChatMessage
		err := c.ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("[CONNECTION] Error reading message: %v", err)
			break
		}

		// Handle the #users command
		if msg.Content == "#users" {
			users, err := c.redisClient.GetActiveUsers()
			if err != nil {
				log.Printf("[CONNECTION] Failed to get active users from Redis: %v", err)
				continue
			}

			response := domain.ChatMessage{
				Sender:    "System",
				Content:   "Active users: " + strings.Join(users, ", "),
				Timestamp: msg.Timestamp,
			}

			c.send <- response
			continue
		}

		// Publish the message to NATS
		err = c.hub.natsClient.Publish("chat.events", msg)
		if err != nil {
			log.Printf("[CONNECTION] Failed to publish message to NATS: %v", err)
		}
	}
}

// writePump listens on the send channel and sends messages to the WebSocket.
func (c *Connection) writePump() {
	defer c.ws.Close()

	for msg := range c.send {
		err := c.ws.WriteJSON(msg)
		if err != nil {
			log.Printf("[CONNECTION] Error sending message: %v", err)
			break
		}
	}
}
