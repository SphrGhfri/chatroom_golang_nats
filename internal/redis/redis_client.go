package redis

import (
	"context"
	"fmt"

	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
	logger logger.Logger
	ctx    context.Context
}

func NewRedisClient(ctx context.Context, redisURL string) (*RedisClient, error) {
	log := logger.FromContext(ctx).WithModule("redis")
	log.Infof("Connecting to Redis at %s", redisURL)

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Errorf("Failed to parse Redis URL: %v", err)
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Monitor context for cleanup
	go func() {
		<-ctx.Done()
		log.Infof("Context cancelled, closing Redis connection")
		client.Close()
	}()

	return &RedisClient{
		client: client,
		logger: log,
		ctx:    ctx,
	}, nil
}

// AddActiveUser adds a user to the active users set.
func (r *RedisClient) AddActiveUser(ctx context.Context, username string) error {
	log := r.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"username": username,
		"action":   "add_active_user",
	})

	log.Infof("Adding active user")
	if err := r.client.SAdd(ctx, "active_users", username).Err(); err != nil {
		log.Errorf("Failed to add active user: %v", err)
		return err
	}
	return nil
}

// RemoveActiveUser removes a user from the active users set.
func (r *RedisClient) RemoveActiveUser(ctx context.Context, username string) error {
	log := r.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"username": username,
		"action":   "remove_active_user",
	})

	log.Infof("Removing active user")
	if err := r.client.SRem(ctx, "active_users", username).Err(); err != nil {
		log.Errorf("Failed to remove active user: %v", err)
		return err
	}
	return nil
}

// GetActiveUsers retrieves all active users.
func (r *RedisClient) GetActiveUsers(ctx context.Context) ([]string, error) {
	log := r.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"action": "get_active_users",
	})

	log.Infof("Retrieving active users")
	users, err := r.client.SMembers(ctx, "active_users").Result()
	if err != nil {
		log.Errorf("Failed to retrieve active users: %v", err)
		return nil, err
	}
	return users, nil
}

// ClearActiveUsers clears all active users from the set.
func (r *RedisClient) ClearActiveUsers(ctx context.Context) error {
	log := r.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"action": "clear_active_users",
	})

	log.Infof("Clearing active users")
	if err := r.client.Del(ctx, "active_users").Err(); err != nil {
		log.Errorf("Failed to clear active users: %v", err)
		return err
	}
	return nil
}

// Check if user exists in active users set
func (r *RedisClient) IsUserActive(ctx context.Context, username string) (bool, error) {
	log := r.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"username": username,
		"action":   "is_user_active",
	})

	log.Infof("Checking if user is active")
	isActive, err := r.client.SIsMember(ctx, "active_users", username).Result()
	if err != nil {
		log.Errorf("Failed to check if user is active: %v", err)
		return false, err
	}
	return isActive, nil
}

// Generic set methods
func (r *RedisClient) SAdd(ctx context.Context, key, member string) error {
	log := r.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"key":    key,
		"member": member,
		"action": "sadd",
	})

	log.Infof("Adding member to set")
	if err := r.client.SAdd(ctx, key, member).Err(); err != nil {
		log.Errorf("Failed to add member to set: %v", err)
		return err
	}
	return nil
}

func (r *RedisClient) SRem(ctx context.Context, key, member string) error {
	log := r.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"key":    key,
		"member": member,
		"action": "srem",
	})

	log.Infof("Removing member from set")
	if err := r.client.SRem(ctx, key, member).Err(); err != nil {
		log.Errorf("Failed to remove member from set: %v", err)
		return err
	}
	return nil
}

func (r *RedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	log := r.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"key":    key,
		"action": "smembers",
	})

	log.Infof("Retrieving members of set")
	members, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		log.Errorf("Failed to retrieve members of set: %v", err)
		return nil, err
	}
	return members, nil
}

// Clear entire Redis
func (r *RedisClient) FlushAll(ctx context.Context) error {
	log := r.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"action": "flush_all",
	})

	log.Infof("Flushing all data")
	if err := r.client.FlushAll(ctx).Err(); err != nil {
		log.Errorf("Failed to flush all data: %v", err)
		return err
	}
	return nil
}

// Close closes the Redis connection.
func (r *RedisClient) Close() error {
	return r.client.Close()
}
