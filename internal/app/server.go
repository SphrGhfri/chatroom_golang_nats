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
	rootCtx     context.Context
	cancel      context.CancelFunc
}

// NewApp initializes and connects all application dependencies
func NewApp(cfg config.Config) (*App, error) {
	// Create application root context
	baseLogger := logger.NewLogger(cfg.LogLevel, cfg.LogFile)
	rootCtx := logger.NewContext(context.Background(), baseLogger)
	rootCtx, rootCancel := context.WithCancel(rootCtx)

	// Get scoped logger for app
	log := logger.FromContext(rootCtx).WithModule("app")
	log.Infof("Initializing application components...")

	// Initialize components with root context
	natsClient, err := nats.NewNATSClient(rootCtx, cfg.NATSURL)
	if err != nil {
		rootCancel()
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	redisClient, err := redis.NewRedisClient(rootCtx, cfg.RedisURL)
	if err != nil {
		rootCancel()
		natsClient.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Initialize chat service
	chatService := service.NewChatService(rootCtx, natsClient, redisClient)

	// Create HTTP server
	httpServer := createHTTPServer(rootCtx, cfg.Port, chatService)

	app := &App{
		cfg:         cfg,
		logger:      log,
		natsClient:  natsClient,
		redisClient: redisClient,
		chatService: chatService,
		httpServer:  httpServer,
		rootCtx:     rootCtx,
		cancel:      rootCancel,
	}

	log.Infof("Application initialized successfully")
	return app, nil
}

func createHTTPServer(ctx context.Context, port int, chatService service.ChatService) *http.Server {
	wsConfig := ws.WSConfig{
		ChatService: chatService,
		RootCtx:     ctx,
	}

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: ws.SetupWebSocketRoutes(wsConfig),
	}
}

// Start runs the application and handles graceful shutdown on signal
func (a *App) Start() error {
	log := a.logger.WithFields(map[string]interface{}{
		"port": a.cfg.Port,
	})

	log.Infof("Starting application server")

	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Fatalf("HTTP server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.WithFields(map[string]interface{}{
		"signal": sig.String(),
	}).Warnf("Received shutdown signal")

	return a.Stop()
}

// Stop gracefully shuts down the server and closes all connections
func (a *App) Stop() error {
	log := a.logger.WithFields(map[string]interface{}{
		"shutdown_timeout": "5s",
	})

	log.Infof("Initiating graceful shutdown")

	// Cancel root context first
	a.cancel()

	// Create shutdown timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(ctx); err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Errorf("HTTP server shutdown error")
	}

	log.Infof("Closing NATS connection")
	a.natsClient.Close()

	log.Infof("Closing Redis connection")
	a.redisClient.Close()

	log.Infof("Shutdown completed successfully")
	return nil
}
