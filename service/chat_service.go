package service

import (
	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
)

// ChatService defines methods for chat functionality.
type ChatService interface {
	PublishMessage(msg domain.ChatMessage) error
	AddActiveUser(username string) error
	RemoveActiveUser(username string) error
	ListActiveUsers() ([]string, error)
}

// chatService implements ChatService.
type chatService struct {
	natsClient  *nats.NATSClient
	redisClient *redis.RedisClient
	logger      logger.Logger
}

// NewChatService constructs a ChatService.
func NewChatService(nc *nats.NATSClient, rc *redis.RedisClient, logg logger.Logger) ChatService {
	return &chatService{
		natsClient:  nc,
		redisClient: rc,
		logger:      logg,
	}
}

// PublishMessage publishes a message to NATS for distribution.
func (c *chatService) PublishMessage(msg domain.ChatMessage) error {
	err := c.natsClient.Publish("chat.events", msg)
	if err != nil {
		c.logger.Errorf("Failed to publish message: %v", err)
	}
	return err
}

func (c *chatService) AddActiveUser(username string) error {
	return c.redisClient.AddActiveUser(username)
}

func (c *chatService) RemoveActiveUser(username string) error {
	return c.redisClient.RemoveActiveUser(username)
}

func (c *chatService) ListActiveUsers() ([]string, error) {
	return c.redisClient.GetActiveUsers()
}
