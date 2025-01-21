package ws

import (
	"log"
	"net/http"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/websocket"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
	gws "github.com/gorilla/websocket"
)

// For demonstration. If you already have your own Upgrader, reuse it.
var upgrader = gws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleWebSocket(
	hub *websocket.Hub,
	chatService service.ChatService,
	logg logger.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logg.Errorf("[WS HANDLER] Upgrade error: %v", err)
			http.Error(w, "Failed to upgrade", http.StatusInternalServerError)
			return
		}

		username := r.URL.Query().Get("username")
		if username == "" {
			logg.Errorf("[WS HANDLER] Missing username param")
			conn.Close()
			return
		}

		// Use ChatService instead of redisClient directly
		if err := chatService.AddActiveUser(username); err != nil {
			log.Printf("[WS HANDLER] Failed to add user '%s': %v", username, err)
		}

		client := &websocket.Connection{
			Ws:          conn,
			Send:        make(chan interface{}, 256),
			Hub:         hub,
			Username:    username,
			ChatService: chatService,
			Logger:      logg,
		}

		hub.Register <- client
		logg.Infof("[WS HANDLER] New connection from %s (user=%s)", conn.RemoteAddr(), username)

		go client.ReadPump()
		go client.WritePump()
	}
}
