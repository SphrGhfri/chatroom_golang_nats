package ws

import (
	"net/http"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/websocket"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
)

func SetupWebSocketRoutes(
	hub *websocket.Hub,
	chatService service.ChatService,
	logg logger.Logger,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", HandleWebSocket(hub, chatService, logg))

	return mux
}
