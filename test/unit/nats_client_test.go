package unit

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/SphrGhfri/chatroom_golang_nats/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
)

func setupNATSClient(t *testing.T) (*nats.NATSClient, context.Context) {
	config := config.MustReadConfig("../../config_test.json")
	baseLogger := logger.NewLogger(config.LogLevel, config.LogFile)
	ctx := logger.NewContext(context.Background(), baseLogger)

	client, err := nats.NewNATSClient(ctx, config.NATSURL)
	assert.NoError(t, err, "Failed to connect to NATS")
	return client, ctx
}

func TestSubscribeRoom(t *testing.T) {
	natsClient, ctx := setupNATSClient(t)
	defer natsClient.Close()

	room := "test_room_subscribe"
	username := "test_user_subscribe"
	messageContent := "Testing SubscribeRoom"
	receivedMessages := make(chan domain.ChatMessage, 1)

	// Subscribe to the room and wait for subscription to be ready
	err := natsClient.SubscribeRoom(ctx, room, username, func(msg domain.ChatMessage) {
		receivedMessages <- msg
	})
	assert.NoError(t, err, "Failed to subscribe to room")
	time.Sleep(100 * time.Millisecond) // Wait for subscription to be fully established

	// Test message
	testMsg := domain.ChatMessage{
		Type:      domain.MessageTypeChat,
		Sender:    "different_user", // Use different sender to avoid filtering
		Content:   messageContent,
		Room:      room,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Publish and wait briefly
	err = natsClient.PublishRoom(ctx, room, testMsg)
	assert.NoError(t, err, "Failed to publish message")
	time.Sleep(100 * time.Millisecond) // Wait for message to be published

	// Wait for message with shorter timeout
	select {
	case msg := <-receivedMessages:
		assert.Equal(t, messageContent, msg.Content)
		assert.Equal(t, "different_user", msg.Sender)
	case <-time.After(1 * time.Second):
		t.Fatal("Did not receive message within timeout")
	}
}

func TestUnsubscribeRoom(t *testing.T) {
	natsClient, ctx := setupNATSClient(t)
	defer natsClient.Close()

	room := "test_room_unsubscribe"
	username := "test_user_unsubscribe"

	// Subscribe first
	err := natsClient.SubscribeRoom(ctx, room, username, func(msg domain.ChatMessage) {})
	assert.NoError(t, err, "Failed to subscribe to room")

	// Unsubscribe and verify
	err = natsClient.UnsubscribeRoom(ctx, room, username)
	assert.NoError(t, err, "Failed to unsubscribe from room")

	// Verify subscription is removed
	subKey := fmt.Sprintf("%s:%s", room, username)
	_, exists := natsClient.SubMapping[subKey]
	assert.False(t, exists, "Subscription should be removed")
}

func TestMultipleUsersInRoom(t *testing.T) {
	natsClient, ctx := setupNATSClient(t)
	defer natsClient.Close()

	room := "test_multi_user_room"
	messageContent := "Multi-user test message"
	received := make(map[string]bool)
	msgChan := make(chan string, 2)

	// Setup two users
	users := []string{"user1", "user2"}
	for _, username := range users {
		err := natsClient.SubscribeRoom(ctx, room, username, func(msg domain.ChatMessage) {
			msgChan <- username
		})
		assert.NoError(t, err, fmt.Sprintf("Failed to subscribe user %s", username))
	}

	// Send test message
	testMsg := domain.ChatMessage{
		Type:      domain.MessageTypeChat,
		Content:   messageContent,
		Room:      room,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	err := natsClient.PublishRoom(ctx, room, testMsg)
	assert.NoError(t, err, "Failed to publish message")

	// Wait for both users to receive the message
	timeout := time.After(2 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case username := <-msgChan:
			received[username] = true
		case <-timeout:
			t.Fatal("Timeout waiting for messages")
		}
	}

	// Verify both users received the message
	for _, username := range users {
		assert.True(t, received[username], fmt.Sprintf("User %s did not receive the message", username))
	}
}

func TestCleanupSubscriptions(t *testing.T) {
	natsClient, ctx := setupNATSClient(t)
	defer natsClient.Close()

	// Create multiple subscriptions
	subscriptions := map[string]string{
		"user1": "room1",
		"user2": "room1",
		"user3": "room2",
	}

	for username, room := range subscriptions {
		err := natsClient.SubscribeRoom(ctx, room, username, func(msg domain.ChatMessage) {})
		assert.NoError(t, err, "Failed to create subscription")
	}

	// Cleanup all subscriptions
	natsClient.CleanupSubscriptions()

	// Verify all subscriptions are removed
	assert.Equal(t, 0, len(natsClient.SubMapping), "All subscriptions should be removed")
}

func TestPublishRoom(t *testing.T) {
	natsClient, ctx := setupNATSClient(t)
	defer natsClient.Close()

	room := "test_publish_room"
	messageContent := "Test Publish Room"

	// Subscribe to the room using correct subject format
	subject := fmt.Sprintf("chat.room.%s", room)
	sub, err := natsClient.Conn.SubscribeSync(subject)
	assert.NoError(t, err, "Failed to subscribe to room")
	defer sub.Unsubscribe()

	// Publish message to the room
	testMsg := domain.ChatMessage{
		Type:      domain.MessageTypeChat,
		Sender:    "room_user",
		Content:   messageContent,
		Room:      room,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	err = natsClient.PublishRoom(ctx, room, testMsg)
	assert.NoError(t, err, "Failed to publish message to room")

	// Receive message
	receivedMsg, err := sub.NextMsg(2 * time.Second)
	assert.NoError(t, err, "Did not receive published message in room")

	var received domain.ChatMessage
	err = json.Unmarshal(receivedMsg.Data, &received)
	assert.NoError(t, err, "Failed to unmarshal received message")

	// Verify message contents
	assert.Equal(t, messageContent, received.Content, "Message content should match")
	assert.Equal(t, room, received.Room, "Room should match")
	assert.Equal(t, domain.MessageTypeChat, received.Type, "Message type should match")
}
