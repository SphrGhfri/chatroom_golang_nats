package nats

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
)

func (c *NATSClient) Publish(subject string, message domain.ChatMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	return c.Conn.Publish(subject, data)
}

// PublishGlobal publishes a message to 'chat.events.global'
func (c *NATSClient) PublishGlobal(msg domain.ChatMessage) error {
	return c.Publish("chat.events.global", msg)
}

// PublishRoom publishes a message to 'chat.events.<roomName>'
func (c *NATSClient) PublishRoom(roomName string, msg domain.ChatMessage) error {
	room := strings.TrimSpace(roomName)
	if room == "" {
		room = "global"
	}
	subject := "chat.events." + room
	return c.Publish(subject, msg)
}
