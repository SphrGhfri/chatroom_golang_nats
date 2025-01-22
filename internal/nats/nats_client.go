package nats

import (
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
)

type NATSClient struct {
	Conn       *nats.Conn
	SubMapping map[string]*nats.Subscription // key: "room:username"
	mu         sync.RWMutex
}

func NewNATSClient(url string) (*NATSClient, error) {
	nc, err := nats.Connect(url, nats.MaxReconnects(-1))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NATSClient{
		Conn:       nc,
		SubMapping: make(map[string]*nats.Subscription),
	}, nil
}

func (c *NATSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, sub := range c.SubMapping {
		sub.Unsubscribe()
	}
	c.Conn.Close()
}
