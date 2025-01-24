package integration

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SphrGhfri/chatroom_golang_nats/api/ws"
	"github.com/SphrGhfri/chatroom_golang_nats/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/redis"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	"github.com/SphrGhfri/chatroom_golang_nats/service"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	conn     *websocket.Conn
	username string
	t        *testing.T // Added t for assertions
}

// Simple setup with single responsibility
func setupTest(t *testing.T) (*httptest.Server, *testClient) {
	config := config.MustReadConfig("../../config_test.json")
	baseLogger := logger.NewLogger(config.LogLevel, config.LogFile)
	ctx := logger.NewContext(context.Background(), baseLogger)

	natsClient, err := nats.NewNATSClient(ctx, config.NATSURL)
	require.NoError(t, err)

	redisClient, err := redis.NewRedisClient(ctx, config.RedisURL)
	require.NoError(t, err)
	redisClient.FlushAll(ctx)

	chatService := service.NewChatService(ctx, natsClient, redisClient)
	server := httptest.NewServer(ws.SetupWebSocketRoutes(ws.WSConfig{
		ChatService: chatService,
		RootCtx:     ctx,
	}))

	// Create first client
	client := connectClient(t, server, "user1")

	t.Cleanup(func() {
		client.conn.Close()
		server.Close()
		redisClient.Close()
		natsClient.Close()
	})

	return server, client
}

// Simple client connection
func connectClient(t *testing.T, server *httptest.Server, username string) *testClient {
	wsURL := "ws" + server.URL[4:] + "/ws?username=" + username
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	return &testClient{conn: conn, username: username, t: t}
}

// Basic send and receive
func (c *testClient) send(msgType domain.MessageType, content, room string) {
	msg := domain.ChatMessage{
		Type:      msgType,
		Content:   content,
		Room:      room,
		Sender:    c.username,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	require.NoError(c.t, c.conn.WriteJSON(msg))
}

func (c *testClient) receive() domain.ChatMessage {
	var msg domain.ChatMessage
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	err := c.conn.ReadJSON(&msg)
	require.NoError(c.t, err)
	return msg
}

func TestMultiUserInteraction(t *testing.T) {
	server, client1 := setupTest(t)
	defer server.Close()

	client2 := connectClient(t, server, "user2")
	_ = client1.receive() // Drain user2 join room messages
	defer client2.conn.Close()

	// Drain welcome messages
	_ = client1.receive()
	_ = client2.receive()

	t.Run("join and chat", func(t *testing.T) {
		// First user joins
		client1.send(domain.MessageTypeJoin, "", "test-room")
		_ = client1.receive() // Drain own join message
		_ = client2.receive()

		// Second user joins
		client2.send(domain.MessageTypeJoin, "", "test-room")
		_ = client1.receive()
		_ = client2.receive() // Drain own join message

		// Chat test - both sender and receiver get the message
		testMsg1 := "Hello from user1"
		client1.send(domain.MessageTypeChat, testMsg1, "")

		msg1 := client2.receive() // client2 gets client1's message
		require.Equal(t, testMsg1, msg1.Content)

		testMsg2 := "Hello from user2"
		client2.send(domain.MessageTypeChat, testMsg2, "")

		msg2 := client1.receive() // client1 gets client2's message
		require.Equal(t, testMsg2, msg2.Content)

	})
}

func TestRoomOperations(t *testing.T) {
	server, client := setupTest(t)
	defer server.Close()

	// Wait for welcome message
	_ = client.receive()

	t.Run("room operations", func(t *testing.T) {
		// Join a room first
		client.send(domain.MessageTypeJoin, "", "test-room")
		_ = client.receive() // Drain join message

		// Test room listing
		client.send(domain.MessageTypeRooms, "", "")
		msg := client.receive()
		require.Equal(t, domain.MessageTypeSystem, msg.Type)
		require.Contains(t, msg.Content, "test-room")

		// Test user listing
		client.send(domain.MessageTypeList, "", "")
		msg = client.receive()
		require.Equal(t, domain.MessageTypeSystem, msg.Type)
		require.Contains(t, msg.Content, "user1")
	})
}
