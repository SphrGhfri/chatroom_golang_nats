package integration

import (
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/SphrGhfri/chatroom_golang_nats/api/ws"
	"github.com/SphrGhfri/chatroom_golang_nats/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/app"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func setupTestServer(t *testing.T) *httptest.Server {
	// Load test configuration from file
	config := config.MustReadConfig("../../config_test.json")
	logg := logger.NewLogger("debug")

	// Create the application instance
	appInstance, err := app.NewApp(config)
	if err != nil {
		t.Fatalf("Failed to initialize the app: %v", err)
	}

	// Start WebSocket hub
	go appInstance.Hub.Run()

	// Create a test HTTP server with WebSocket routes
	server := httptest.NewServer(ws.SetupWebSocketRoutes(appInstance.Hub, appInstance.ChatService, logg))

	t.Cleanup(func() {
		server.Close()
		appInstance.Hub.Close()

		// Flush Redis data after each test
		err := appInstance.RedisClient.FlushAll()
		assert.NoError(t, err, "Failed to flush Redis after test")

		appInstance.NatsClient.Close()
		appInstance.RedisClient.Close()
	})

	return server
}

func TestWebSocketConnection(t *testing.T) {
	server := setupTestServer(t)
	wsURL := "ws" + server.URL[4:] + "/ws?username=testuser"

	conn, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err, "Failed to connect to WebSocket server")
	defer conn.Close()

	message := domain.ChatMessage{
		Type:    domain.MessageTypeChat,
		Sender:  "testuser",
		Content: "Hello, WebSocket!",
		Room:    "global",
	}

	err = conn.WriteJSON(message)
	assert.NoError(t, err, "Failed to send WebSocket message")

	// Read response
	var receivedMsg domain.ChatMessage
	err = conn.ReadJSON(&receivedMsg)
	assert.NoError(t, err, "Failed to read WebSocket message")
	assert.Equal(t, "Hello, WebSocket!", receivedMsg.Content)
	assert.Equal(t, "testuser", receivedMsg.Sender)
}

func TestWebSocketBroadcast(t *testing.T) {
	server := setupTestServer(t)
	wsURL := "ws" + server.URL[4:] + "/ws?username=user1"

	// Connect first user
	conn1, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err, "Failed to connect first user")
	defer conn1.Close()

	// Connect second user
	wsURL2 := "ws" + server.URL[4:] + "/ws?username=user2"
	conn2, _, err := gws.DefaultDialer.Dial(wsURL2, nil)
	assert.NoError(t, err, "Failed to connect second user")
	defer conn2.Close()

	time.Sleep(500 * time.Millisecond)

	// Send message from user1
	message := domain.ChatMessage{
		Type:    domain.MessageTypeChat,
		Sender:  "user1",
		Content: "Hello from user1",
		Room:    "global",
	}
	err = conn1.WriteJSON(message)
	assert.NoError(t, err, "Failed to send message from user1")

	time.Sleep(500 * time.Millisecond)

	// Read message from user2
	var receivedMsg domain.ChatMessage
	err = conn2.ReadJSON(&receivedMsg)
	assert.NoError(t, err, "Failed to receive message on user2")

	assert.Equal(t, "Hello from user1", receivedMsg.Content)
	assert.Equal(t, "user1", receivedMsg.Sender)
}

func TestWebSocketRoomFunctionality(t *testing.T) {
	server := setupTestServer(t)

	// Connect user1 to room1
	wsURL := "ws" + server.URL[4:] + "/ws?username=user1"
	conn1, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn1.Close()

	// Connect user2 to room1
	wsURL2 := "ws" + server.URL[4:] + "/ws?username=user2"
	conn2, _, err := gws.DefaultDialer.Dial(wsURL2, nil)
	assert.NoError(t, err)
	defer conn2.Close()

	// User1 joins room1
	joinMessage := domain.ChatMessage{
		Type: domain.MessageTypeJoin,
		Room: "room1",
	}
	err = conn1.WriteJSON(joinMessage)
	assert.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	// User2 joins room1
	joinMessage2 := domain.ChatMessage{
		Type: domain.MessageTypeJoin,
		Room: "room1",
	}
	err = conn2.WriteJSON(joinMessage2)
	assert.NoError(t, err)

	// Wait to ensure room joins are processed
	time.Sleep(500 * time.Millisecond)

	// Discard system messages (user1 and user2 join notifications)
	var systemMsg domain.ChatMessage
	for i := 0; i < 1; i++ {
		err = conn2.ReadJSON(&systemMsg)
		assert.NoError(t, err, "Failed to read system message for user2")
		assert.Contains(t, systemMsg.Content, "joined room room1", "Expected join confirmation")
	}

	// Send message in room1
	message := domain.ChatMessage{
		Type:    domain.MessageTypeChat,
		Sender:  "user1",
		Content: "Hello Room1",
		Room:    "room1",
	}
	err = conn1.WriteJSON(message)
	assert.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	// Set a read deadline to avoid infinite wait
	conn2.SetReadDeadline(time.Now().Add(3 * time.Second))

	// Read actual chat message from user1
	var receivedMsg domain.ChatMessage
	err = conn2.ReadJSON(&receivedMsg)
	assert.NoError(t, err, "User2 should receive the message")
	assert.Equal(t, "Hello Room1", receivedMsg.Content)
	assert.Equal(t, "user1", receivedMsg.Sender)
}

func TestWebSocketUserLeavesRoom(t *testing.T) {
	server := setupTestServer(t)

	wsURL := "ws" + server.URL[4:] + "/ws?username=user1"
	conn1, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn1.Close()

	// Join room1
	joinMessage := domain.ChatMessage{
		Type: domain.MessageTypeJoin,
		Room: "room1",
	}
	err = conn1.WriteJSON(joinMessage)
	assert.NoError(t, err)

	// Allow some time for room join processing
	time.Sleep(500 * time.Millisecond)

	// Leave room1
	leaveMessage := domain.ChatMessage{
		Type: domain.MessageTypeLeave,
		Room: "room1",
	}
	err = conn1.WriteJSON(leaveMessage)
	assert.NoError(t, err)

	// Allow some time for leave processing
	time.Sleep(500 * time.Millisecond)

	// Expect two messages: leaving room1 and joining global
	var leftMsg domain.ChatMessage
	err = conn1.ReadJSON(&leftMsg)
	assert.NoError(t, err)
	assert.Contains(t, leftMsg.Content, "user1 joined room room1", "Expected join old room message")

	var joinedMsg domain.ChatMessage
	err = conn1.ReadJSON(&joinedMsg)
	assert.NoError(t, err)
	assert.Contains(t, joinedMsg.Content, "joined room global", "Expected join global message")
}

func TestConcurrentWebSocketConnections(t *testing.T) {
	server := setupTestServer(t)
	wsURL := "ws" + server.URL[4:] + "/ws"

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u := fmt.Sprintf("%s?username=user%d", wsURL, i)
			conn, _, err := gws.DefaultDialer.Dial(u, nil)
			assert.NoError(t, err)
			defer conn.Close()

			// Send a message from each connection
			msg := domain.ChatMessage{
				Type:    domain.MessageTypeChat,
				Sender:  fmt.Sprintf("user%d", i),
				Content: fmt.Sprintf("Hello from user%d", i),
				Room:    "global",
			}
			err = conn.WriteJSON(msg)
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()
}
