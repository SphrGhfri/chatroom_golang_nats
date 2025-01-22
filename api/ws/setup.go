package ws

import (
	"net/http"

	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
)

type WSConfig struct {
	ChatService service.ChatService
	Logger      logger.Logger
}

func SetupWebSocketRoutes(cfg WSConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", HandleWebSocket(cfg.ChatService, cfg.Logger))
	return mux
}
