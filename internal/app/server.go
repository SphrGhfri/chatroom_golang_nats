package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SphrGhfri/chatroom_golang_nats/api/ws"
	"github.com/SphrGhfri/chatroom_golang_nats/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
)

// App represents the main application structure holding all dependencies
type App struct {
	cfg         config.Config
	logger      logger.Logger
	natsClient  *nats.NATSClient
	redisClient *redis.RedisClient
	chatService service.ChatService
	httpServer  *http.Server
}

// NewApp initializes and connects all application dependencies
func NewApp(cfg config.Config) (*App, error) {
	logg := logger.NewLogger(cfg.LogLevel)

	natsClient, err := nats.NewNATSClient(cfg.NATSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	redisClient, err := redis.NewRedisClient(cfg.RedisURL)
	if err != nil {
		natsClient.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	if err := redisClient.FlushAll(); err != nil {
		logg.Errorf("Failed to clear redis: %v", err)
	}

	chatService := service.NewChatService(natsClient, redisClient, logg)

	app := &App{
		cfg:         cfg,
		logger:      logg,
		natsClient:  natsClient,
		redisClient: redisClient,
		chatService: chatService,
		httpServer:  createHTTPServer(cfg, chatService, logg),
	}

	return app, nil
}

// createHTTPServer sets up the HTTP server with WebSocket routes
func createHTTPServer(cfg config.Config, chatService service.ChatService, logger logger.Logger) *http.Server {
	wsConfig := ws.WSConfig{
		ChatService: chatService,
		Logger:      logger,
	}

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: ws.SetupWebSocketRoutes(wsConfig),
	}
}

// Start runs the application and handles graceful shutdown on signal
func (a *App) Start() error {
	go func() {
		a.logger.Infof("Starting HTTP server on port %d", a.cfg.Port)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Fatalf("HTTP server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	a.logger.Infof("Received signal %s, shutting down gracefully...", sig)
	return a.Stop()
}

// Stop gracefully shuts down the server and closes all connections
func (a *App) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.logger.Errorf("HTTP server shutdown error: %v", err)
	}

	a.natsClient.Close()
	a.redisClient.Close()

	a.logger.Infof("Application stopped.")
	return nil
}
