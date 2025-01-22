package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
)

// ChatService defines the interface
type ChatService interface {
	PublishMessage(msg domain.ChatMessage) error
	AddActiveUser(username string) error
	RemoveActiveUser(username string) error
	ListActiveUsers() ([]string, error)

	JoinRoom(roomName, username string, msgHandler func(domain.ChatMessage)) error
	LeaveRoom(roomName, username string) error
	ListRoomMembers(roomName string) ([]string, error)
	ListAllRooms() ([]string, error)
	SwitchRoom(oldRoom, newRoom, username string, msgHandler func(domain.ChatMessage)) error
	IsUserActive(username string) (bool, error)
}

type chatService struct {
	natsClient  *nats.NATSClient
	redisClient *redis.RedisClient
	logger      logger.Logger
}

func NewChatService(nc *nats.NATSClient, rc *redis.RedisClient, logg logger.Logger) ChatService {
	return &chatService{
		natsClient:  nc,
		redisClient: rc,
		logger:      logg,
	}
}

// If msg.Room is empty => publish to chat.events.global
func (c *chatService) PublishMessage(msg domain.ChatMessage) error {
	room := strings.TrimSpace(msg.Room)
	err := c.natsClient.PublishRoom(room, msg)

	if err != nil {
		c.logger.Errorf("PublishMessage error: %v", err)
	}
	return err
}

// Presence
func (c *chatService) AddActiveUser(username string) error {
	return c.redisClient.AddActiveUser(username)
}
func (c *chatService) RemoveActiveUser(username string) error {
	return c.redisClient.RemoveActiveUser(username)
}
func (c *chatService) ListActiveUsers() ([]string, error) {
	return c.redisClient.GetActiveUsers()
}
func (c *chatService) IsUserActive(username string) (bool, error) {
	exists, err := c.redisClient.IsUserActive(username)
	if err != nil {
		c.logger.Errorf("Failed to check user existence: %v", err)
		return false, err
	}
	return exists, nil
}

// Rooms
func (c *chatService) JoinRoom(roomName, username string, msgHandler func(domain.ChatMessage)) error {
	if roomName == "" || username == "" {
		return fmt.Errorf("room name and username cannot be empty")
	}

	// Add user to Redis room
	if err := c.redisClient.SAdd("room:"+roomName, username); err != nil {
		return fmt.Errorf("failed to add user to room: %w", err)
	}

	// Track room in all_rooms
	if err := c.redisClient.SAdd("all_rooms", roomName); err != nil {
		return fmt.Errorf("failed to track room: %w", err)
	}

	// Subscribe to NATS topic with the provided message handler
	if err := c.natsClient.SubscribeRoom(roomName, username, func(msg domain.ChatMessage) {
		// Don't send the message back to the sender
		if msg.Sender != username {
			msgHandler(msg)
		}
	}); err != nil {
		return fmt.Errorf("failed to subscribe to NATS: %w", err)
	}

	// Notify room members
	c.PublishMessage(domain.ChatMessage{
		Type:      domain.MessageTypeSystem,
		Content:   fmt.Sprintf("%s joined the room", username),
		Room:      roomName,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	})

	return nil
}

func (c *chatService) LeaveRoom(roomName, username string) error {
	if roomName == "" || username == "" {
		return fmt.Errorf("room name and username cannot be empty")
	}

	// Notify room members before leaving
	c.PublishMessage(domain.ChatMessage{
		Type:      domain.MessageTypeSystem,
		Content:   fmt.Sprintf("%s left the room", username),
		Room:      roomName,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	})

	// First unsubscribe from NATS
	if err := c.natsClient.UnsubscribeRoom(roomName, username); err != nil {
		c.logger.Errorf("failed to unsubscribe from room %s: %v", roomName, err)
		// Continue execution - we still want to remove from Redis
	}

	// Remove user from the Redis room set
	key := "room:" + roomName
	if err := c.redisClient.SRem(key, username); err != nil {
		return fmt.Errorf("failed to remove user from room in Redis: %w", err)
	}

	// Check if room is empty
	members, err := c.redisClient.SMembers(key)
	if err != nil {
		c.logger.Errorf("failed to get room members: %v", err)
	} else if len(members) == 0 {
		// Room is empty, remove it from all_rooms
		if err := c.redisClient.SRem("all_rooms", roomName); err != nil {
			c.logger.Errorf("failed to remove empty room from all_rooms: %v", err)
		}
	}

	c.logger.Infof("%s left room %s", username, roomName)
	return nil
}

func (c *chatService) ListRoomMembers(roomName string) ([]string, error) {
	return c.redisClient.SMembers("room:" + roomName)
}
func (c *chatService) ListAllRooms() ([]string, error) {
	return c.redisClient.SMembers("all_rooms")
}

func (c *chatService) SwitchRoom(oldRoom, newRoom, username string, msgHandler func(domain.ChatMessage)) error {
	if err := c.LeaveRoom(oldRoom, username); err != nil {
		return fmt.Errorf("failed to leave old room: %w", err)
	}

	if err := c.JoinRoom(newRoom, username, msgHandler); err != nil {
		// Try to rejoin old room on failure
		_ = c.JoinRoom(oldRoom, username, msgHandler)
		return fmt.Errorf("failed to join new room: %w", err)
	}

	return nil
}
