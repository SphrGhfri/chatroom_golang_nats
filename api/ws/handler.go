package ws

import (
	"log"
	"net/http"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/websocket"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
	gws "github.com/gorilla/websocket"
)

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

		// Check if username is already taken
		existingUsers, err := chatService.ListActiveUsers()
		if err != nil {
			logg.Errorf("[WS HANDLER] Error listing active users: %v", err)
			conn.Close()
			return
		}
		for _, u := range existingUsers {
			if u == username {
				logg.Errorf("[WS HANDLER] Username '%s' already in use!", username)

				_ = conn.WriteJSON(map[string]string{
					"type":    "username_exists",
					"content": "Username is already exists. Please reconnect with a new username.",
				})

				conn.Close()
				return
			}
		}

		if err := chatService.AddActiveUser(username); err != nil {
			log.Printf("[WS HANDLER] Failed to add user '%s': %v", username, err)
		}

		// Place user in "global" by default
		_ = chatService.JoinRoom("global", username)

		client := &websocket.Connection{
			Ws:          conn,
			Send:        make(chan interface{}, 256),
			Hub:         hub,
			Username:    username,
			ChatService: chatService,
			Logger:      logg,
			CurrentRoom: "global",
		}

		hub.Register <- client
		logg.Infof("[WS HANDLER] New connection from %s (user=%s)", conn.RemoteAddr(), username)

		go client.ReadPump()
		go client.WritePump()
	}
}
