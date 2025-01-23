package nats

import (
	"encoding/json"
	"fmt"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
)

// PublishRoom broadcasts a message to all subscribers in a specific room
// Messages are JSON encoded before publishing
// Uses subject format "chat.room.<roomName>" for NATS routing
func (c *NATSClient) PublishRoom(roomName string, msg domain.ChatMessage) error {
	// Format the subject to match subscription pattern
	subject := fmt.Sprintf("chat.room.%s", roomName)

	// Serialize message to JSON for transmission
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish to all subscribers in the room
	return c.Conn.Publish(subject, data)
}
