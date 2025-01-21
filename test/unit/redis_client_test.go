package unit

import (
	"os"
	"testing"

	"github.com/SphrGhfri/chatroom_golang_nats/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/stretchr/testify/assert"
)

var redisClient *redis.RedisClient

func TestMain(m *testing.M) {
	config := config.MustReadConfig("../../config_test.json")
	var err error
	redisClient, err = redis.NewRedisClient(config.RedisURL)
	if err != nil {
		panic("Failed to connect to Redis for tests: " + err.Error())
	}

	code := m.Run()

	// Clean up after all tests
	_ = redisClient.FlushAll()
	redisClient.Close()

	os.Exit(code)
}

// Helper function to clear Redis before each test
func clearRedis() {
	_ = redisClient.FlushAll()
}

func TestAddActiveUser(t *testing.T) {
	clearRedis() // Ensure fresh start
	err := redisClient.AddActiveUser("user1")
	assert.Nil(t, err)

	users, err := redisClient.GetActiveUsers()
	assert.Nil(t, err)
	assert.Contains(t, users, "user1")
}

func TestRemoveActiveUser(t *testing.T) {
	clearRedis()
	_ = redisClient.AddActiveUser("user2")
	err := redisClient.RemoveActiveUser("user2")
	assert.Nil(t, err)

	users, err := redisClient.GetActiveUsers()
	assert.Nil(t, err)
	assert.NotContains(t, users, "user2")
}

func TestGetActiveUsers(t *testing.T) {
	clearRedis()
	_ = redisClient.AddActiveUser("user3")
	_ = redisClient.AddActiveUser("user4")

	users, err := redisClient.GetActiveUsers()
	assert.Nil(t, err)
	assert.ElementsMatch(t, []string{"user3", "user4"}, users)
}

func TestClearActiveUsers(t *testing.T) {
	clearRedis()
	_ = redisClient.AddActiveUser("user5")
	err := redisClient.ClearActiveUsers()
	assert.Nil(t, err)

	users, err := redisClient.GetActiveUsers()
	assert.Nil(t, err)
	assert.Empty(t, users)
}

func TestSAddAndSMembers(t *testing.T) {
	clearRedis()
	err := redisClient.SAdd("room:testroom", "user1")
	assert.Nil(t, err)

	members, err := redisClient.SMembers("room:testroom")
	assert.Nil(t, err)
	assert.Contains(t, members, "user1")
}

func TestSRem(t *testing.T) {
	clearRedis()
	_ = redisClient.SAdd("room:testroom", "user2")
	err := redisClient.SRem("room:testroom", "user2")
	assert.Nil(t, err)

	members, err := redisClient.SMembers("room:testroom")
	assert.Nil(t, err)
	assert.NotContains(t, members, "user2")
}

func TestFlushAll(t *testing.T) {
	clearRedis()
	_ = redisClient.SAdd("room:testroom", "user3")
	err := redisClient.FlushAll()
	assert.Nil(t, err)

	members, err := redisClient.SMembers("room:testroom")
	assert.Nil(t, err)
	assert.Empty(t, members)
}
