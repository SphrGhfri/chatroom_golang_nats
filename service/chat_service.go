package service

import (
	"fmt"
	"strings"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/websocket"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
)

// ChatService defines the interface
type ChatService interface {
	PublishMessage(msg domain.ChatMessage) error
	AddActiveUser(username string) error
	RemoveActiveUser(username string) error
	ListActiveUsers() ([]string, error)

	JoinRoom(roomName, username string) error
	LeaveRoom(roomName, username string) error
	ListRoomMembers(roomName string) ([]string, error)
	ListAllRooms() ([]string, error)
	SwitchRoom(oldRoom, newRoom, username string) error
}

type chatService struct {
	natsClient  *nats.NATSClient
	redisClient *redis.RedisClient
	hub         *websocket.Hub
	logger      logger.Logger
}

func NewChatService(nc *nats.NATSClient, rc *redis.RedisClient, hub *websocket.Hub, logg logger.Logger) ChatService {
	return &chatService{
		natsClient:  nc,
		redisClient: rc,
		hub:         hub,
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

// Rooms
func (c *chatService) JoinRoom(roomName, username string) error {
	if roomName == "" || username == "" {
		return fmt.Errorf("room name and username cannot be empty")
	}

	// Subscribe to the room **only once** (if not already)
	err := c.natsClient.SubscribeRoom(roomName, username, func(msg domain.ChatMessage) {
		c.logger.Infof("Received message in room %s: %s", msg.Room, msg.Content)
		c.forwardToHub(msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to room %s: %w", roomName, err)
	}

	// Add user to the Redis room set
	if err := c.redisClient.SAdd("room:"+roomName, username); err != nil {
		return err
	}

	// Ensure the room is tracked in 'all_rooms'
	if err := c.redisClient.SAdd("all_rooms", roomName); err != nil {
		return err
	}

	c.logger.Infof("%s joined room %s", username, roomName)
	return nil
}

func (c *chatService) LeaveRoom(roomName, username string) error {
	if roomName == "" || username == "" {
		return fmt.Errorf("room name and username cannot be empty")
	}

	// Remove user from the Redis room set
	key := "room:" + roomName
	err := c.redisClient.SRem(key, username)
	if err != nil {
		return err
	}

	// Check if there are no more members in the room.
	members, _ := c.redisClient.SMembers(key)
	if len(members) == 0 {
		// Optional: remove the room from the 'all_rooms' set
		_ = c.redisClient.SRem("all_rooms", roomName)
		// Unsubscribe from NATS for this room since it's empty.
		if err := c.natsClient.UnsubscribeRoom(roomName); err != nil {
			c.logger.Errorf("Error unsubscribing room %s: %v", roomName, err)
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

func (c *chatService) SwitchRoom(oldRoom, newRoom, username string) error {
	if oldRoom != "" {
		if err := c.LeaveRoom(oldRoom, username); err != nil {
			return err
		}
	}
	return c.JoinRoom(newRoom, username)
}

func (c *chatService) forwardToHub(msg domain.ChatMessage) {
	// Just stick it on the broadcast channel:
	c.hub.Broadcast <- msg
}
