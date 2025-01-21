package nats

import (
	"encoding/json"
	"fmt"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/nats-io/nats.go"
)

// Subscribe user to a specific room
func (c *NATSClient) SubscribeRoom(roomName, username string, handleFunc func(domain.ChatMessage)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	subject := fmt.Sprintf("chat.events.%s", roomName)

	// If the room is already subscribed, do nothing.
	if _, exists := c.SubMapping[roomName]; exists {
		return nil
	}

	sub, err := c.Conn.Subscribe(subject, func(msg *nats.Msg) {
		var chatMsg domain.ChatMessage
		if err := json.Unmarshal(msg.Data, &chatMsg); err != nil {
			fmt.Printf("Failed to deserialize message: %v\n", err)
			return
		}
		// Ensure the room is set (you could also use chatMsg.Room if it’s already set)
		chatMsg.Room = roomName
		handleFunc(chatMsg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to room %s: %w", roomName, err)
	}

	c.SubMapping[roomName] = sub
	return nil
}

func (c *NATSClient) UnsubscribeRoom(roomName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if sub, exists := c.SubMapping[roomName]; exists {
		if err := sub.Unsubscribe(); err != nil {
			return fmt.Errorf("failed to unsubscribe room %s: %w", roomName, err)
		}
		delete(c.SubMapping, roomName)
	}
	return nil
}
