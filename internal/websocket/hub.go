package websocket

import (
	"sync"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
)

type Hub struct {
	mu         sync.RWMutex
	clients    map[*Connection]bool
	Register   chan *Connection
	Unregister chan *Connection
	Broadcast  chan domain.ChatMessage
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Connection]bool),
		Broadcast:  make(chan domain.ChatMessage),
		Register:   make(chan *Connection),
		Unregister: make(chan *Connection),
	}
}

// Run starts the Hub's main loop for handling connections and broadcasts.
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.Register:
			h.addClient(conn)
		case conn := <-h.Unregister:
			h.removeClient(conn)
		case msg := <-h.Broadcast:
			h.broadcastMessage(msg)
		}
	}
}

// Close gracefully shuts down the Hub, closing all connections.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		close(conn.Send)
		conn.Ws.Close()
		delete(h.clients, conn)
	}
}

// addClient adds a new connection to the Hub.
func (h *Hub) addClient(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
}

// removeClient removes a connection from the Hub.
func (h *Hub) removeClient(conn *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.clients[conn]; exists {
		delete(h.clients, conn)
		close(conn.Send)
	}
}

// broadcastMessage sends a message to all active clients.
func (h *Hub) broadcastMessage(msg domain.ChatMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		if msg.Room != "" && conn.CurrentRoom != msg.Room {
			continue
		}
		select {
		case conn.Send <- msg:
		default:
			h.removeClient(conn)
		}
	}
}
