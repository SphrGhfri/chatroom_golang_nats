package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisClient(redisURL string) (*RedisClient, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)
	ctx := context.Background()

	// Check connection
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{client: client, ctx: ctx}, nil
}

// AddActiveUser adds a user to the active users set.
func (r *RedisClient) AddActiveUser(username string) error {
	return r.client.SAdd(r.ctx, "active_users", username).Err()
}

// RemoveActiveUser removes a user from the active users set.
func (r *RedisClient) RemoveActiveUser(username string) error {
	return r.client.SRem(r.ctx, "active_users", username).Err()
}

// GetActiveUsers retrieves all active users.
func (r *RedisClient) GetActiveUsers() ([]string, error) {
	return r.client.SMembers(r.ctx, "active_users").Result()
}

// ClearActiveUsers clears all active users from the set.
func (r *RedisClient) ClearActiveUsers() error {
	return r.client.Del(r.ctx, "active_users").Err()
}

// Check if user exists in active users set
func (r *RedisClient) IsUserActive(username string) (bool, error) {
	return r.client.SIsMember(r.ctx, "active_users", username).Result()
}

// Generic set methods
func (r *RedisClient) SAdd(key, member string) error {
	return r.client.SAdd(r.ctx, key, member).Err()
}
func (r *RedisClient) SRem(key, member string) error {
	return r.client.SRem(r.ctx, key, member).Err()
}
func (r *RedisClient) SMembers(key string) ([]string, error) {
	return r.client.SMembers(r.ctx, key).Result()
}

// Clear entire Redis
func (r *RedisClient) FlushAll() error {
	return r.client.FlushAll(r.ctx).Err()
}

// Close closes the Redis connection.
func (r *RedisClient) Close() error {
	return r.client.Close()
}
