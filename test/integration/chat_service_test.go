package integration

import (
	"testing"
	"time"

	"github.com/SphrGhfri/chatroom_golang_nats/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/websocket"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
	"github.com/stretchr/testify/assert"
)

func setupChatService(t *testing.T) service.ChatService {
	config := config.MustReadConfig("../../config_test.json")
	redisClient, err := redis.NewRedisClient(config.RedisURL)
	assert.NoError(t, err, "Failed to connect to Redis")

	natsClient, err := nats.NewNATSClient(config.NATSURL)
	assert.NoError(t, err, "Failed to connect to NATS")

	hub := websocket.NewHub()
	chatService := service.NewChatService(natsClient, redisClient, hub, logger.NewLogger("debug"))

	// Ensure Redis is flushed before the test starts
	err = redisClient.FlushAll()
	assert.NoError(t, err, "Failed to flush Redis before test")

	t.Cleanup(func() {
		redisClient.FlushAll()
		redisClient.Close()
		natsClient.Close()
	})

	return chatService
}

// Test AddActiveUser and ListActiveUsers
func TestActiveUserManagement(t *testing.T) {
	chatService := setupChatService(t)

	// Add users
	assert.NoError(t, chatService.AddActiveUser("user1"))
	assert.NoError(t, chatService.AddActiveUser("user2"))

	// Check active users
	users, err := chatService.ListActiveUsers()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"user1", "user2"}, users)

	// Remove user
	assert.NoError(t, chatService.RemoveActiveUser("user1"))

	// Check remaining users
	users, err = chatService.ListActiveUsers()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"user2"}, users)
}

// Test JoinRoom and ListRoomMembers
func TestRoomManagement(t *testing.T) {
	chatService := setupChatService(t)

	// Join users to room
	assert.NoError(t, chatService.JoinRoom("roomA", "user1"))
	assert.NoError(t, chatService.JoinRoom("roomA", "user2"))

	// Verify room members
	members, err := chatService.ListRoomMembers("roomA")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"user1", "user2"}, members)

	// Leave room
	assert.NoError(t, chatService.LeaveRoom("roomA", "user1"))

	// Verify updated members
	members, err = chatService.ListRoomMembers("roomA")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"user2"}, members)

	// Ensure empty room is removed from tracking
	assert.NoError(t, chatService.LeaveRoom("roomA", "user2"))

	rooms, err := chatService.ListAllRooms()
	assert.NoError(t, err)
	assert.Empty(t, rooms)
}

// Test PublishMessage
func TestPublishMessage(t *testing.T) {
	chatService := setupChatService(t)

	message := domain.ChatMessage{
		Type:      domain.MessageTypeChat,
		Sender:    "user1",
		Content:   "Hello, World!",
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Room:      "testRoom",
	}

	assert.NoError(t, chatService.JoinRoom("testRoom", "user1"))

	// Create a channel to capture published messages
	receivedMessages := make(chan domain.ChatMessage, 1)

	// Simulate a subscription by listening for messages
	go func() {
		_ = chatService.JoinRoom("testRoom", "user1")
		err := chatService.PublishMessage(message)
		assert.NoError(t, err)
		receivedMessages <- message
	}()

	select {
	case received := <-receivedMessages:
		assert.Equal(t, message.Content, received.Content)
		assert.Equal(t, message.Sender, received.Sender)
		assert.Equal(t, "testRoom", received.Room)
	case <-time.After(5 * time.Second):
		t.Error("Did not receive message in time")
	}
}
