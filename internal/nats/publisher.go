package nats

import (
	"encoding/json"
	"fmt"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
)

func (c *NATSClient) PublishRoom(roomName string, msg domain.ChatMessage) error {
	subject := fmt.Sprintf("chat.room.%s", roomName)
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return c.Conn.Publish(subject, data)
}
