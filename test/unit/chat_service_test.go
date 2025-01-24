package unit

import (
	"context"
	"testing"

	"github.com/SphrGhfri/chatroom_golang_nats/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
	"github.com/stretchr/testify/assert"
)

func setupChatService(t *testing.T) (service.ChatService, context.Context) {
	config := config.MustReadConfig("../../config_test.json")
	baseLogger := logger.NewLogger(config.LogLevel, config.LogFile)
	ctx := logger.NewContext(context.Background(), baseLogger)

	natsClient, err := nats.NewNATSClient(ctx, config.NATSURL)
	assert.NoError(t, err)

	redisClient, err := redis.NewRedisClient(ctx, config.RedisURL)
	assert.NoError(t, err)
	redisClient.FlushAll(ctx)

	chatService := service.NewChatService(ctx, natsClient, redisClient)

	t.Cleanup(func() {
		redisClient.FlushAll(ctx)
		redisClient.Close()
		natsClient.Close()
	})

	return chatService, ctx
}

// Test AddActiveUser and ListActiveUsers
func TestActiveUserManagement(t *testing.T) {
	chatService, ctx := setupChatService(t)

	// Add users
	assert.NoError(t, chatService.AddActiveUser(ctx, "user1"))
	assert.NoError(t, chatService.AddActiveUser(ctx, "user2"))

	// Check active users
	users, err := chatService.ListActiveUsers(ctx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"user1", "user2"}, users)

	// Remove user
	assert.NoError(t, chatService.RemoveActiveUser(ctx, "user1"))

	// Check remaining users
	users, err = chatService.ListActiveUsers(ctx)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"user2"}, users)
}

// Test JoinRoom and ListRoomMembers
func TestRoomManagement(t *testing.T) {
	chatService, ctx := setupChatService(t)
	msgChan := make(chan domain.ChatMessage, 1)

	// Join users to room with message handler
	handler := func(msg domain.ChatMessage) {
		msgChan <- msg
	}

	assert.NoError(t, chatService.JoinRoom(ctx, "roomA", "user1", handler))
	assert.NoError(t, chatService.JoinRoom(ctx, "roomA", "user2", handler))

	// Verify room members
	members, err := chatService.ListRoomMembers(ctx, "roomA")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"user1", "user2"}, members)

	// Leave room
	assert.NoError(t, chatService.LeaveRoom(ctx, "roomA", "user1"))

	// Verify updated members
	members, err = chatService.ListRoomMembers(ctx, "roomA")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"user2"}, members)

	// Clean up final room
	assert.NoError(t, chatService.LeaveRoom(ctx, "roomA", "user2"))

	// Verify room is removed
	rooms, err := chatService.ListAllRooms(ctx)
	assert.NoError(t, err)
	assert.Empty(t, rooms)
}

func TestIsUserActive(t *testing.T) {
	chatService, ctx := setupChatService(t)

	// Check non-existent user
	exists, err := chatService.IsUserActive(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Add user and check
	assert.NoError(t, chatService.AddActiveUser(ctx, "testuser"))
	exists, err = chatService.IsUserActive(ctx, "testuser")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Remove user and check again
	assert.NoError(t, chatService.RemoveActiveUser(ctx, "testuser"))
	exists, err = chatService.IsUserActive(ctx, "testuser")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestSwitchRoom(t *testing.T) {
	chatService, ctx := setupChatService(t)

	// Create handler just for tracking membership
	messageHandler := func(msg domain.ChatMessage) {}

	// Initial room join
	err := chatService.JoinRoom(ctx, "room1", "user1", messageHandler)
	assert.NoError(t, err)

	// Switch to new room
	err = chatService.SwitchRoom(ctx, "room1", "room2", "user1", messageHandler)
	assert.NoError(t, err)

	// Verify user is no longer in old room
	members1, err := chatService.ListRoomMembers(ctx, "room1")
	assert.NoError(t, err)
	assert.NotContains(t, members1, "user1")

	// Verify user is in new room
	members2, err := chatService.ListRoomMembers(ctx, "room2")
	assert.NoError(t, err)
	assert.Contains(t, members2, "user1")
}
