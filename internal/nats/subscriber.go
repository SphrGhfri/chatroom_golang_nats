package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/nats-io/nats.go"
)

// SubscribeRoom subscribes a user to a specific chat room
// and filters out their own messages to prevent echo
// subKey format: "roomName:username" is used to track unique subscriptions
func (c *NATSClient) SubscribeRoom(ctx context.Context, roomName, username string, handleFunc func(domain.ChatMessage)) error {
	log := c.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"room":     roomName,
		"username": username,
	})

	// Lock to prevent concurrent modification of subscription map
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create NATS subject using room name (e.g., "chat.room.general")
	subject := fmt.Sprintf("chat.room.%s", roomName)
	// Create unique subscription key to track user's room subscription
	subKey := fmt.Sprintf("%s:%s", roomName, username)

	// Prevent duplicate subscriptions for same user in same room
	if _, exists := c.SubMapping[subKey]; exists {
		log.Infof("User already subscribed to room")
		return nil
	}

	log.Infof("Subscribing user to room")
	// Create subscription with message handler
	sub, err := c.Conn.Subscribe(subject, func(msg *nats.Msg) {
		// Decode incoming message
		var chatMsg domain.ChatMessage
		if err := json.Unmarshal(msg.Data, &chatMsg); err != nil {
			log.Errorf("Failed to unmarshal message: %v", err)
			return // Skip invalid messages
		}
		// Only process messages from other users
		if chatMsg.Sender != username {
			handleFunc(chatMsg)
		}
	})
	if err != nil {
		log.Errorf("Failed to subscribe to room: %v", err)
		return fmt.Errorf("failed to subscribe to room %s: %w", roomName, err)
	}

	c.SubMapping[subKey] = sub
	log.Infof("Successfully subscribed to room")
	return nil
}

// UnsubscribeRoom removes a user's subscription from a specific room
// If the subscription doesn't exist, it returns nil
// This ensures clean removal from both NATS server and local mapping
func (c *NATSClient) UnsubscribeRoom(ctx context.Context, roomName, username string) error {
	log := c.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"room":     roomName,
		"username": username,
	})

	c.mu.Lock()
	defer c.mu.Unlock()

	subKey := fmt.Sprintf("%s:%s", roomName, username)
	if sub, exists := c.SubMapping[subKey]; exists {
		log.Infof("Unsubscribing from room")
		if err := sub.Unsubscribe(); err != nil {
			log.Errorf("Failed to unsubscribe: %v", err)
			return fmt.Errorf("failed to unsubscribe: %w", err)
		}
		delete(c.SubMapping, subKey)
		log.Infof("Successfully unsubscribed from room")
	}
	return nil
}

// CleanupSubscriptions removes all active subscriptions for this client
// Used during shutdown or when needing to reset all subscriptions
// Ignores unsubscribe errors to ensure complete cleanup
func (c *NATSClient) CleanupSubscriptions() {
	c.mu.Lock()
	defer c.mu.Unlock()

	log := c.logger.WithFields(map[string]interface{}{
		"sub_count": len(c.SubMapping),
	})

	log.Infof("Cleaning up all subscriptions")
	for room, sub := range c.SubMapping {
		if err := sub.Unsubscribe(); err != nil {
			log.Errorf("Failed to unsubscribe from %s: %v", room, err)
		}
		delete(c.SubMapping, room)
	}
	log.Infof("Subscription cleanup completed")
}
