package unit

import (
	"testing"

	"github.com/SphrGhfri/chatroom_golang_nats/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
	"github.com/stretchr/testify/assert"
)

func setupChatService(t *testing.T) (service.ChatService, *nats.NATSClient, *redis.RedisClient) {
	config := config.MustReadConfig("../../config_test.json")

	natsClient, err := nats.NewNATSClient(config.NATSURL)
	assert.NoError(t, err)

	redisClient, err := redis.NewRedisClient(config.RedisURL)
	assert.NoError(t, err)
	redisClient.FlushAll()

	chatService := service.NewChatService(natsClient, redisClient, logger.NewLogger("debug"))

	t.Cleanup(func() {
		redisClient.FlushAll()
		redisClient.Close()
		natsClient.Close()
	})

	return chatService, natsClient, redisClient
}

// Test AddActiveUser and ListActiveUsers
func TestActiveUserManagement(t *testing.T) {
	chatService, _, _ := setupChatService(t)

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
	chatService, _, _ := setupChatService(t)
	msgChan := make(chan domain.ChatMessage, 1)

	// Join users to room with message handler
	handler := func(msg domain.ChatMessage) {
		msgChan <- msg
	}

	assert.NoError(t, chatService.JoinRoom("roomA", "user1", handler))
	assert.NoError(t, chatService.JoinRoom("roomA", "user2", handler))

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

	// Clean up final room
	assert.NoError(t, chatService.LeaveRoom("roomA", "user2"))

	// Verify room is removed
	rooms, err := chatService.ListAllRooms()
	assert.NoError(t, err)
	assert.Empty(t, rooms)
}

func TestIsUserActive(t *testing.T) {
	chatService, _, _ := setupChatService(t)

	// Check non-existent user
	exists, err := chatService.IsUserActive("nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Add user and check
	assert.NoError(t, chatService.AddActiveUser("testuser"))
	exists, err = chatService.IsUserActive("testuser")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Remove user and check again
	assert.NoError(t, chatService.RemoveActiveUser("testuser"))
	exists, err = chatService.IsUserActive("testuser")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestSwitchRoom(t *testing.T) {
	chatService, _, _ := setupChatService(t)

	// Create handler just for tracking membership
	messageHandler := func(msg domain.ChatMessage) {}

	// Initial room join
	err := chatService.JoinRoom("room1", "user1", messageHandler)
	assert.NoError(t, err)

	// Switch to new room
	err = chatService.SwitchRoom("room1", "room2", "user1", messageHandler)
	assert.NoError(t, err)

	// Verify user is no longer in old room
	members1, err := chatService.ListRoomMembers("room1")
	assert.NoError(t, err)
	assert.NotContains(t, members1, "user1")

	// Verify user is in new room
	members2, err := chatService.ListRoomMembers("room2")
	assert.NoError(t, err)
	assert.Contains(t, members2, "user1")
}
