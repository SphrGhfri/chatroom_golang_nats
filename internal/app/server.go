package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/websocket"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
)

func RunServer(cfg config.Config, log logger.Logger, natsClient *nats.NATSClient) error {
	// Initialize WebSocket hub
	hub := websocket.NewHub(natsClient)
	go hub.Run()

	// Start HTTP server with WebSocket routes
	httpServer := setupHTTPServer(cfg, log, hub)

	// Start NATS subscriber
	if err := setupNATSSubscriber(natsClient, log, hub); err != nil {
		return fmt.Errorf("failed to set up NATS subscriber: %w", err)
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		log.Infof("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Errorf("Failed to gracefully shut down HTTP server: %v", err)
		}

		// Close WebSocket hub and NATS client
		hub.Close()
		natsClient.Close()
	}()

	// Start the server
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Infof("Server is running on %s", addr)
	return httpServer.ListenAndServe()
}

func setupHTTPServer(cfg config.Config, log logger.Logger, hub *websocket.Hub) *http.Server {
	mux := http.NewServeMux()

	// WebSocket route
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Infof("Handling WebSocket request from %s", r.RemoteAddr)
		websocket.ServeWS(hub, w, r)
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
