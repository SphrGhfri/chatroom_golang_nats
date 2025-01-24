package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
)

// PublishRoom broadcasts a message to all subscribers in a specific room
// Messages are JSON encoded before publishing
// Uses subject format "chat.room.<roomName>" for NATS routing
func (c *NATSClient) PublishRoom(ctx context.Context, roomName string, msg domain.ChatMessage) error {
	log := c.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"room":     roomName,
		"sender":   msg.Sender,
		"msg_type": msg.Type,
	})

	// Format the subject to match subscription pattern
	subject := fmt.Sprintf("chat.room.%s", roomName)

	// Serialize message to JSON for transmission
	data, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("Failed to marshal message: %v", err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	log.Infof("Publishing message to room")
	// Publish to all subscribers in the room
	if err := c.Conn.Publish(subject, data); err != nil {
		log.Errorf("Failed to publish message: %v", err)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}
