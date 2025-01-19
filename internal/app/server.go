package app

import (
	"fmt"
	"net/http"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/websocket"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
)

func RunServer(cfg config.Config, log logger.Logger, natsClient *nats.NATSClient) error {
	// Initialize Redis client
	redisClient, err := redis.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize WebSocket hub
	hub := websocket.NewHub(natsClient)
	go hub.Run()

	// Start HTTP server with WebSocket routes
	httpServer := setupHTTPServer(cfg, log, hub, redisClient)

	// Start NATS subscriber
	if err := setupNATSSubscriber(natsClient, log, hub); err != nil {
		return fmt.Errorf("failed to set up NATS subscriber: %w", err)
	}

	// Start the server
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Infof("Server is running on %s", addr)
	return httpServer.ListenAndServe()
}

func setupHTTPServer(cfg config.Config, log logger.Logger, hub *websocket.Hub, redisClient *redis.RedisClient) *http.Server {
	mux := http.NewServeMux()

	// WebSocket route
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Infof("Handling WebSocket request from %s", r.RemoteAddr)
		websocket.ServeWS(hub, redisClient, w, r)
	})

	// Return configured HTTP server
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}
}

func setupNATSSubscriber(natsClient *nats.NATSClient, log logger.Logger, hub *websocket.Hub) error {
	return natsClient.Subscribe("chat.events", func(msg domain.ChatMessage) {
		log.Infof("Received message from NATS: %+v", msg)
		hub.Broadcast(msg)
	})
}
