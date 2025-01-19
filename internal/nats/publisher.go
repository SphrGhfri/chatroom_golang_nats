package nats

import (
	"encoding/json"
	"fmt"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
)

func (c *NATSClient) Publish(subject string, message domain.ChatMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	return c.Conn.Publish(subject, data)
}
