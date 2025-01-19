package websocket

import (
	"log"
	"net/http"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for testing; restrict in production.
	},
}

// Connection represents a single WebSocket connection to a client.
type Connection struct {
	ws   *websocket.Conn         // WebSocket connection
	send chan domain.ChatMessage // Channel for sending messages to the client
	hub  *Hub                    // Reference to the Hub managing this connection
}

// ServeWS upgrades an HTTP connection to a WebSocket connection and registers it with the Hub.
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[CONNECTION] Failed to upgrade WebSocket: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusInternalServerError)
		return
	}

	client := &Connection{
		ws:   conn,
		send: make(chan domain.ChatMessage, 256), // Buffered channel for outgoing messages
		hub:  hub,
	}

	// Register the client with the Hub
	hub.register <- client
	log.Printf("[CONNECTION] New WebSocket connection: %s", conn.RemoteAddr())

	// Start the read and write pumps
	go client.readPump()
	go client.writePump()
}

// readPump listens for incoming messages from the WebSocket and forwards them to the Hub.
func (c *Connection) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.ws.Close()
	}()

	for {
		var msg domain.ChatMessage
		err := c.ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("[CONNECTION] Error reading message: %v", err)
			break
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
