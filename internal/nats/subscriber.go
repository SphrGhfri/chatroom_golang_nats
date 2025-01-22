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

	subject := fmt.Sprintf("chat.room.%s", roomName)
	subKey := fmt.Sprintf("%s:%s", roomName, username)

	// If this specific user is already subscribed to this room, do nothing
	if _, exists := c.SubMapping[subKey]; exists {
		return nil
	}

	sub, err := c.Conn.Subscribe(subject, func(msg *nats.Msg) {
		var chatMsg domain.ChatMessage
		if err := json.Unmarshal(msg.Data, &chatMsg); err != nil {
			return
		}
		// Filter out messages from the same user at NATS level
		if chatMsg.Sender != username {
			handleFunc(chatMsg)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to room %s: %w", roomName, err)
	}

	c.SubMapping[subKey] = sub
	return nil
}

func (c *NATSClient) UnsubscribeRoom(roomName, username string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	subKey := fmt.Sprintf("%s:%s", roomName, username)
	if sub, exists := c.SubMapping[subKey]; exists {
		if err := sub.Unsubscribe(); err != nil {
			return fmt.Errorf("failed to unsubscribe: %w", err)
		}
		delete(c.SubMapping, subKey)
	}
	return nil
}

// CleanupSubscriptions unsubscribes from all rooms
func (c *NATSClient) CleanupSubscriptions() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for room, sub := range c.SubMapping {
		_ = sub.Unsubscribe()
		delete(c.SubMapping, room)
	}
}
