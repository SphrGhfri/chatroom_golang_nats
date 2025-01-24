package service

import (
	"context"
	"fmt"
	"time"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
)

// ChatService defines the interface
type ChatService interface {
	PublishMessage(ctx context.Context, msg domain.ChatMessage) error
	AddActiveUser(ctx context.Context, username string) error
	RemoveActiveUser(ctx context.Context, username string) error
	ListActiveUsers(ctx context.Context) ([]string, error)

	JoinRoom(ctx context.Context, roomName, username string, msgHandler func(domain.ChatMessage)) error
	LeaveRoom(ctx context.Context, roomName, username string) error
	ListRoomMembers(ctx context.Context, roomName string) ([]string, error)
	ListAllRooms(ctx context.Context) ([]string, error)
	SwitchRoom(ctx context.Context, oldRoom, newRoom, username string, msgHandler func(domain.ChatMessage)) error
	IsUserActive(ctx context.Context, username string) (bool, error)
}

type chatService struct {
	natsClient  *nats.NATSClient
	redisClient *redis.RedisClient
	logger      logger.Logger
	ctx         context.Context // Add context
}

func NewChatService(ctx context.Context, nc *nats.NATSClient, rc *redis.RedisClient) ChatService {
	log := logger.FromContext(ctx).WithModule("chat")
	return &chatService{
		natsClient:  nc,
		redisClient: rc,
		logger:      log,
		ctx:         ctx,
	}
}

// publish to chat.events.<room>
func (c *chatService) PublishMessage(ctx context.Context, msg domain.ChatMessage) error {
	log := c.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"room":   msg.Room,
		"sender": msg.Sender,
		"type":   msg.Type,
	})

	log.Infof("Publishing message to room")
	if err := c.natsClient.PublishRoom(ctx, msg.Room, msg); err != nil {
		log.Errorf("Failed to publish message: %v", err)
		return err
	}
	return nil
}

// Presence
func (c *chatService) AddActiveUser(ctx context.Context, username string) error {
	return c.redisClient.AddActiveUser(ctx, username)
}
func (c *chatService) RemoveActiveUser(ctx context.Context, username string) error {
	return c.redisClient.RemoveActiveUser(ctx, username)
}
func (c *chatService) ListActiveUsers(ctx context.Context) ([]string, error) {
	return c.redisClient.GetActiveUsers(ctx)
}
func (c *chatService) IsUserActive(ctx context.Context, username string) (bool, error) {
	exists, err := c.redisClient.IsUserActive(ctx, username)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("Failed to check user existence: %v", err)
		return false, err
	}
	return exists, nil
}

// Rooms
func (c *chatService) JoinRoom(ctx context.Context, roomName, username string, msgHandler func(domain.ChatMessage)) error {
	log := c.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"room":     roomName,
		"username": username,
	})

	if roomName == "" || username == "" {
		log.Errorf("Invalid room name or username")
		return fmt.Errorf("room name and username cannot be empty")
	}

	log.Infof("User joining room")

	// Add user to Redis room
	if err := c.redisClient.SAdd(ctx, "room:"+roomName, username); err != nil {
		log.Errorf("Failed to add user to room: %v", err)
		return fmt.Errorf("failed to add user to room: %w", err)
	}

	// Track room in all_rooms
	if err := c.redisClient.SAdd(ctx, "all_rooms", roomName); err != nil {
		log.Errorf("Failed to track room: %v", err)
		return fmt.Errorf("failed to track room: %w", err)
	}

	// Subscribe to NATS topic with the provided message handler
	if err := c.natsClient.SubscribeRoom(ctx, roomName, username, func(msg domain.ChatMessage) {
		// Don't send the message back to the sender
		if msg.Sender != username {
			msgHandler(msg)
		}
	}); err != nil {
		log.Errorf("Failed to subscribe to NATS: %v", err)
		return fmt.Errorf("failed to subscribe to NATS: %w", err)
	}

	// Notify room members
	c.PublishMessage(ctx, domain.ChatMessage{
		Type:      domain.MessageTypeSystem,
		Content:   fmt.Sprintf("%s joined the room %s", username, roomName),
		Room:      roomName,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	})

	return nil
}

func (c *chatService) LeaveRoom(ctx context.Context, roomName, username string) error {
	log := c.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"room":     roomName,
		"username": username,
	})

	if roomName == "" || username == "" {
		log.Errorf("Invalid room name or username")
		return fmt.Errorf("room name and username cannot be empty")
	}

	// Notify room members before leaving
	c.PublishMessage(ctx, domain.ChatMessage{
		Type:      domain.MessageTypeSystem,
		Content:   fmt.Sprintf("%s left the room", username),
		Room:      roomName,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	})

	// First unsubscribe from NATS
	if err := c.natsClient.UnsubscribeRoom(ctx, roomName, username); err != nil {
		log.Errorf("failed to unsubscribe from room %s: %v", roomName, err)
		// Continue execution - we still want to remove from Redis
	}

	// Remove user from the Redis room set
	key := "room:" + roomName
	if err := c.redisClient.SRem(ctx, key, username); err != nil {
		log.Errorf("Failed to remove user from room in Redis: %v", err)
		return fmt.Errorf("failed to remove user from room in Redis: %w", err)
	}

	// Check if room is empty
	members, err := c.redisClient.SMembers(ctx, key)
	if err != nil {
		log.Errorf("failed to get room members: %v", err)
	} else if len(members) == 0 {
		// Room is empty, remove it from all_rooms
		if err := c.redisClient.SRem(ctx, "all_rooms", roomName); err != nil {
			log.Errorf("failed to remove empty room from all_rooms: %v", err)
		}
	}

	log.Infof("%s left room %s", username, roomName)
	return nil
}

func (c *chatService) ListRoomMembers(ctx context.Context, roomName string) ([]string, error) {
	return c.redisClient.SMembers(ctx, "room:"+roomName)
}
func (c *chatService) ListAllRooms(ctx context.Context) ([]string, error) {
	return c.redisClient.SMembers(ctx, "all_rooms")
}

func (c *chatService) SwitchRoom(ctx context.Context, oldRoom, newRoom, username string, msgHandler func(domain.ChatMessage)) error {
	if err := c.LeaveRoom(ctx, oldRoom, username); err != nil {
		return fmt.Errorf("failed to leave old room: %w", err)
	}

	if err := c.JoinRoom(ctx, newRoom, username, msgHandler); err != nil {
		// Try to rejoin old room on failure
		_ = c.JoinRoom(ctx, oldRoom, username, msgHandler)
		return fmt.Errorf("failed to join new room: %w", err)
	}

	return nil
}
