package nats

import (
	"context"
	"fmt"
	"sync"

	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/nats-io/nats.go"
)

// NATSClient handles NATS connection and subscriptions
type NATSClient struct {
	Conn       *nats.Conn
	SubMapping map[string]*nats.Subscription // Stores active subscriptions by room:username
	mu         sync.RWMutex                  // Protects concurrent access to SubMapping
	logger     logger.Logger                 // Logger for NATS operations
	ctx        context.Context
}

// NewNATSClient creates a new NATS client with persistent connection
func NewNATSClient(ctx context.Context, url string) (*NATSClient, error) {
	// Get logger from context and set module
	log := logger.FromContext(ctx).WithModule("nats")
	log.Infof("Connecting to NATS server at %s", url)

	nc, err := nats.Connect(url, nats.MaxReconnects(-1))
	if err != nil {
		log.Errorf("Failed to connect to NATS: %v", err)
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Monitor context for cleanup
	go func() {
		<-ctx.Done()
		log.Infof("Context cancelled, closing NATS connection")
		nc.Close()
	}()

	client := &NATSClient{
		Conn:       nc,
		SubMapping: make(map[string]*nats.Subscription),
		logger:     log,
		ctx:        ctx,
	}

	return client, nil
}

// Close unsubscribes from all topics and closes the NATS connection
func (c *NATSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, sub := range c.SubMapping {
		sub.Unsubscribe()
	}
	c.Conn.Close()
}
