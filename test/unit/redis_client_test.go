package unit

import (
	"os"
	"testing"

	"context"

	"github.com/SphrGhfri/chatroom_golang_nats/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/stretchr/testify/assert"
)

var (
	redisClient *redis.RedisClient
	testCtx     context.Context
)

func TestMain(m *testing.M) {
	config := config.MustReadConfig("../../config_test.json")
	baseLogger := logger.NewLogger(config.LogLevel, config.LogFile)
	testCtx = logger.NewContext(context.Background(), baseLogger)

	var err error
	redisClient, err = redis.NewRedisClient(testCtx, config.RedisURL)
	if err != nil {
		panic("Failed to connect to Redis for tests: " + err.Error())
	}

	code := m.Run()

	// Clean up after all tests
	redisClient.FlushAll(testCtx)
	redisClient.Close()

	os.Exit(code)
}

// Helper function to clear Redis before each test
func clearRedis() {
	redisClient.FlushAll(testCtx)
}

func TestAddActiveUser(t *testing.T) {
	clearRedis() // Ensure fresh start
	err := redisClient.AddActiveUser(testCtx, "user1")
	assert.Nil(t, err)

	users, err := redisClient.GetActiveUsers(testCtx)
	assert.Nil(t, err)
	assert.Contains(t, users, "user1")
}

func TestRemoveActiveUser(t *testing.T) {
	clearRedis()
	_ = redisClient.AddActiveUser(testCtx, "user2")
	err := redisClient.RemoveActiveUser(testCtx, "user2")
	assert.Nil(t, err)

	users, err := redisClient.GetActiveUsers(testCtx)
	assert.Nil(t, err)
	assert.NotContains(t, users, "user2")
}

func TestGetActiveUsers(t *testing.T) {
	clearRedis()
	_ = redisClient.AddActiveUser(testCtx, "user3")
	_ = redisClient.AddActiveUser(testCtx, "user4")

	users, err := redisClient.GetActiveUsers(testCtx)
	assert.Nil(t, err)
	assert.ElementsMatch(t, []string{"user3", "user4"}, users)
}

func TestClearActiveUsers(t *testing.T) {
	clearRedis()
	_ = redisClient.AddActiveUser(testCtx, "user5")
	err := redisClient.ClearActiveUsers(testCtx)
	assert.Nil(t, err)

	users, err := redisClient.GetActiveUsers(testCtx)
	assert.Nil(t, err)
	assert.Empty(t, users)
}

func TestSAddAndSMembers(t *testing.T) {
	clearRedis()
	err := redisClient.SAdd(testCtx, "room:testroom", "user1")
	assert.Nil(t, err)

	members, err := redisClient.SMembers(testCtx, "room:testroom")
	assert.Nil(t, err)
	assert.Contains(t, members, "user1")
}

func TestSRem(t *testing.T) {
	clearRedis()
	_ = redisClient.SAdd(testCtx, "room:testroom", "user2")
	err := redisClient.SRem(testCtx, "room:testroom", "user2")
	assert.Nil(t, err)

	members, err := redisClient.SMembers(testCtx, "room:testroom")
	assert.Nil(t, err)
	assert.NotContains(t, members, "user2")
}

func TestFlushAll(t *testing.T) {
	clearRedis()
	_ = redisClient.SAdd(testCtx, "room:testroom", "user3")
	err := redisClient.FlushAll(testCtx)
	assert.Nil(t, err)

	members, err := redisClient.SMembers(testCtx, "room:testroom")
	assert.Nil(t, err)
	assert.Empty(t, members)
}
