package nats

import (
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
)

type NATSClient struct {
	Conn       *nats.Conn
	SubMapping map[string]*nats.Subscription // Store subscriptions per username
	mu         sync.Mutex
}

func NewNATSClient(url string) (*NATSClient, error) {
	// Connect to NATS
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NATSClient{
		Conn:       nc,
		SubMapping: make(map[string]*nats.Subscription),
	}, nil
}

func (c *NATSClient) Close() {
	c.Conn.Close()
}
