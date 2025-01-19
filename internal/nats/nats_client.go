package nats

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

type NATSClient struct {
	Conn *nats.Conn
}

func NewNATSClient(url string) (*NATSClient, error) {
	// Connect to NATS
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NATSClient{
		Conn: nc,
	}, nil
}

func (c *NATSClient) Close() {
	c.Conn.Close()
}
