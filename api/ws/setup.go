package ws

import (
	"context"
	"net/http"

	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
)

type WSConfig struct {
	ChatService service.ChatService
	RootCtx     context.Context
}

func SetupWebSocketRoutes(cfg WSConfig) http.Handler {
	mux := http.NewServeMux()
	// Get logger from context for websocket module
	log := logger.FromContext(cfg.RootCtx).WithModule("websocket")
	mux.HandleFunc("/ws", HandleWebSocket(cfg.ChatService, cfg.RootCtx, log))
	return mux
}
