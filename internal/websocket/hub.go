package websocket

import (
	"sync"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
)

// Hub manages all WebSocket connections and broadcasts messages.
type Hub struct {
	mu         sync.RWMutex            // Mutex for safe access to the clients map
	clients    map[*Connection]bool    // Active WebSocket connections
	broadcast  chan domain.ChatMessage // Channel for broadcasting messages
	register   chan *Connection        // Channel for new connections
	unregister chan *Connection        // Channel for disconnecting clients
	natsClient *nats.NATSClient        // NATS client for message publishing
}

// NewHub creates a new Hub instance.
func NewHub(natsClient *nats.NATSClient) *Hub {
	return &Hub{
		clients:    make(map[*Connection]bool),
		broadcast:  make(chan domain.ChatMessage),
		register:   make(chan *Connection),
		unregister: make(chan *Connection),
		natsClient: natsClient,
	}
}

// Run starts the Hub's main loop for handling connections and broadcasts.
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.addClient(conn)
		case conn := <-h.unregister:
			h.removeClient(conn)
		case msg := <-h.broadcast:
			h.broadcastMessage(msg)
		}
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
		close(conn.send)
	}
}

// broadcastMessage sends a message to all active clients.
func (h *Hub) broadcastMessage(msg domain.ChatMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- msg:
		default:
			h.removeClient(client)
		}
	}
}

// Broadcast queues a message for broadcasting.
func (h *Hub) Broadcast(msg domain.ChatMessage) {
	h.broadcast <- msg
}
