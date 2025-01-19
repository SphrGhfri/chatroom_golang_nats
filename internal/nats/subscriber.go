package nats

import (
	"encoding/json"
	"fmt"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/nats-io/nats.go"
)

func (c *NATSClient) Subscribe(subject string, handleFunc func(domain.ChatMessage)) error {
	_, err := c.Conn.Subscribe(subject, func(msg *nats.Msg) {
		var chatMessage domain.ChatMessage
		if err := json.Unmarshal(msg.Data, &chatMessage); err != nil {
			fmt.Printf("Failed to deserialize message: %v\n", err)
			return
		}

		// Handle the received message
		handleFunc(chatMessage)
	})

	return err
}
