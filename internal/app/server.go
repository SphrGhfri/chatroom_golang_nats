package app

import (
	"fmt"
	"net/http"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/websocket"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
)

// RunServer initializes and runs the WebSocket server and NATS subscriber.
func RunServer(cfg config.Config, log logger.Logger, natsClient *nats.NATSClient) error {
	// Initialize WebSocket hub
	hub := websocket.NewHub(natsClient)
	go hub.Run()

	// Set up HTTP server with WebSocket routes
	httpServer := setupHTTPServer(cfg, log, hub)

	// Log server start and begin listening
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Infof("Server is running on %s", addr)

	return httpServer.ListenAndServe()
}

// setupHTTPServer sets up the HTTP server with WebSocket endpoints.
func setupHTTPServer(cfg config.Config, log logger.Logger, hub *websocket.Hub) *http.Server {
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Infof("Handling WebSocket request from %s", r.RemoteAddr)
		websocket.ServeWS(hub, w, r)
	})

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}
}
