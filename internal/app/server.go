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
	"github.com/SphrGhfri/chatroom_golang_nats/internal/websocket"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
)

type App struct {
	Cfg         config.Config
	Logger      logger.Logger
	NatsClient  *nats.NATSClient
	RedisClient *redis.RedisClient
	ChatService service.ChatService
	Hub         *websocket.Hub
	HTTPServer  *http.Server
}

// NewApp creates the entire application with dependencies.
func NewApp(cfg config.Config) (*App, error) {
	// 1) Logger
	logg := logger.NewLogger(cfg.LogLevel)

	// 2) NATS
	natsClient, err := nats.NewNATSClient(cfg.NATSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// 3) Redis
	redisClient, err := redis.NewRedisClient(cfg.RedisURL)
	if err != nil {
		natsClient.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// 4) Clear old active users
	if err := redisClient.FlushAll(); err != nil {
		logg.Errorf("Failed to clear redis: %v", err)
	}

	// 5) Create the Hub (for WebSocket connections)
	hub := websocket.NewHub()

	// 6) Create ChatService (Business Logic)
	chatService := service.NewChatService(natsClient, redisClient, hub, logg)

	// 7) Create the HTTP server (with routes)
	httpServer := createHTTPServer(cfg, logg, hub, chatService)

	// 8) Create the app container
	app := &App{
		Cfg:         cfg,
		Logger:      logg,
		NatsClient:  natsClient,
		RedisClient: redisClient,
		ChatService: chatService,
		Hub:         hub,
		HTTPServer:  httpServer,
	}

	return app, nil
}

// createHTTPServer sets up the HTTP routes and returns an *http.Server.
func createHTTPServer(
	cfg config.Config,
	logg logger.Logger,
	hub *websocket.Hub,
	chatSvc service.ChatService,
) *http.Server {

	wsMux := ws.SetupWebSocketRoutes(hub, chatSvc, logg)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: wsMux,
	}
	return srv
}

// Start runs application: starts the Hub, NATS subscriber, and the HTTP server.
func (a *App) Start() error {
	// 1) Start the Hub in a separate goroutine
	go a.Hub.Run()

	a.Logger.Infof("Hub started.")

	// 3) Start HTTP server (async)
	go func() {
		a.Logger.Infof("Starting HTTP server on port %d", a.Cfg.Port)
		if err := a.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.Logger.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 4) Listen for OS signals to gracefully shut down
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	a.Logger.Infof("Received signal %s, shutting down gracefully...", sig)
	return a.Stop()
}

// Stop closes resources (HTTP, NATS, Redis) gracefully.
func (a *App) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1) Shut down HTTP server
	if err := a.HTTPServer.Shutdown(ctx); err != nil {
		a.Logger.Errorf("HTTP server shutdown error: %v", err)
	}

	// 2) Close the Hub (which closes all connections)
	a.Hub.Close()

	// 3) Close NATS and Redis
	a.NatsClient.Close()
	a.RedisClient.Close()

	a.Logger.Infof("Application stopped.")
	return nil
}
