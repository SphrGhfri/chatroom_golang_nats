package main

import (
	"flag"
	"os"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/app"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
)

var configPath = flag.String("config", "config.json", "service configuration file")

func main() {
	flag.Parse()

	// Load configuration
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

	// Start the server
	if err := app.RunServer(cfg, logg, natsClient); err != nil {
		logg.Fatalf("Server error: %v", err)
	}
}
