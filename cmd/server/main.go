package main

import (
	"flag"
	"os"
	"time"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
)

var configPath = flag.String("config", "config.json", "service configuration file")

func main() {
	flag.Parse()

	if v := os.Getenv("CONFIG_PATH"); len(v) > 0 {
		*configPath = v
	}

	cfg := config.MustReadConfig(*configPath)

	// Initialize logger
	logg := logger.NewLogger(cfg.LogLevel)

	// Initialize NATS client
	natsClient, err := nats.NewNATSClient(cfg.NATSURL)
	if err != nil {
		logg.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	logg.Infof("Successfully connected to NATS: %s", cfg.NATSURL)

	// Publish a test message
	testMessage := domain.ChatMessage{
		Sender:    "test_user",
		Content:   "Hello, NATS!",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := natsClient.Publish("chat.events", testMessage); err != nil {
		logg.Errorf("Failed to publish test message: %v", err)
	} else {
		logg.Infof("Test message published to NATS subject: chat.events")
	}

	// Subscribe to NATS
	err = natsClient.Subscribe("chat.events", func(msg domain.ChatMessage) {
		logg.Infof("Received message from NATS: %+v", msg)
	})

	if err != nil {
		logg.Errorf("Failed to subscribe to NATS: %v", err)
		return
	}

	logg.Infof("Server starting on port %d", cfg.Port)
}
