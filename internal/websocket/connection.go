package websocket

import (
	"log"
	"net/http"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/gorilla/websocket"
)

// Upgrader for WebSocket connections
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for testing; restrict in production.
	},
}

// Connection represents a single WebSocket client.
type Connection struct {
	ws     *websocket.Conn         // WebSocket connection
	send   chan domain.ChatMessage // Channel for sending messages to the client
	hub    *Hub                    // Reference to the Hub
	closed chan struct{}           // Signal channel for closing
}

// ServeWS upgrades HTTP to WebSocket and registers the connection with the Hub.
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusInternalServerError)
		return
	}

	conn := &Connection{
		ws:     ws,
		send:   make(chan domain.ChatMessage, 256),
		hub:    hub,
		closed: make(chan struct{}),
	}

	hub.register <- conn
	log.Printf("New WebSocket connection: %s", ws.RemoteAddr())

	go conn.readMessages()
	go conn.writeMessages()
}

// readMessages handles incoming messages from the WebSocket client.
func (c *Connection) readMessages() {
	defer c.close()

	for {
		var msg domain.ChatMessage
		if err := c.ws.ReadJSON(&msg); err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		// Publish the message to NATS
		if err := c.hub.natsClient.Publish("chat.events", msg); err != nil {
			log.Printf("Failed to publish message to NATS: %v", err)
		}
	}
}

// writeMessages sends messages to the WebSocket client.
func (c *Connection) writeMessages() {
	defer c.close()

	for {
		select {
		case msg := <-c.send:
			if err := c.ws.WriteJSON(msg); err != nil {
				log.Printf("Error sending message: %v", err)
				return
			}
		case <-c.closed:
			return
		}
	}
}

// close unregisters the connection and closes resources.
func (c *Connection) close() {
	c.hub.unregister <- c
	c.ws.Close()
	close(c.closed)
}
