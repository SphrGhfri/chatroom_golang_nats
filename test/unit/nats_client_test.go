package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/SphrGhfri/chatroom_golang_nats/config"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/nats"
)

func setupNATSClient(t *testing.T) *nats.NATSClient {
	config := config.MustReadConfig("../../config_test.json")
	client, err := nats.NewNATSClient(config.NATSURL)
	assert.Nil(t, err, "Failed to connect to NATS")
	return client
}

func TestSubscribeRoom(t *testing.T) {
	natsClient := setupNATSClient(t)
	defer natsClient.Close()

	room := "test_room_subscribe"
	username := "test_user_subscribe"
	messageContent := "Testing SubscribeRoom"

	receivedMessages := make(chan domain.ChatMessage, 1)

	// Subscribe to the room
	err := natsClient.SubscribeRoom(room, username, func(msg domain.ChatMessage) {
		receivedMessages <- msg
	})
	assert.Nil(t, err, "Failed to subscribe to room")

	// Publish test message
	testMsg := domain.ChatMessage{Type: domain.MessageTypeChat, Sender: username, Content: messageContent, Room: room}
	_ = natsClient.PublishRoom(room, testMsg)

	select {
	case msg := <-receivedMessages:
		assert.Equal(t, messageContent, msg.Content, "Received message content should match")
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive message")
	}
}

func TestUnsubscribeRoom(t *testing.T) {
	natsClient := setupNATSClient(t)
	defer natsClient.Close()

	room := "test_room_unsubscribe"
	username := "test_user_unsubscribe"

	// Subscribe to the room
	err := natsClient.SubscribeRoom(room, username, func(msg domain.ChatMessage) {})
	assert.Nil(t, err, "Failed to subscribe to room")

	// Unsubscribe from the room
	err = natsClient.UnsubscribeRoom(room)
	assert.Nil(t, err, "Failed to unsubscribe from room")

	// Verify subscription is removed
	_, exists := natsClient.SubMapping[room]
	assert.False(t, exists, "Room should be unsubscribed")
}

func TestPublish(t *testing.T) {
	natsClient := setupNATSClient(t)
	defer natsClient.Close()

	subject := "chat.events.test_publish"
	messageContent := "Test Publish"

	// Subscribe to the subject
	sub, err := natsClient.Conn.SubscribeSync(subject)
	assert.Nil(t, err, "Failed to subscribe to subject")
	defer sub.Unsubscribe()

	// Publish message
	testMsg := domain.ChatMessage{Type: domain.MessageTypeChat, Sender: "publisher", Content: messageContent}
	err = natsClient.Publish(subject, testMsg)
	assert.Nil(t, err, "Failed to publish message")

	// Receive message
	receivedMsg, err := sub.NextMsg(2 * time.Second)
	assert.Nil(t, err, "Did not receive published message")

	var received domain.ChatMessage
	err = json.Unmarshal(receivedMsg.Data, &received)
	assert.Nil(t, err, "Failed to unmarshal received message")
	assert.Equal(t, messageContent, received.Content, "Published message content should match")
}

func TestPublishGlobal(t *testing.T) {
	natsClient := setupNATSClient(t)
	defer natsClient.Close()

	messageContent := "Test Publish Global"
	sub, err := natsClient.Conn.SubscribeSync("chat.events.global")
	assert.Nil(t, err, "Failed to subscribe to global")

	// Publish to global
	testMsg := domain.ChatMessage{Type: domain.MessageTypeChat, Sender: "global_user", Content: messageContent, Room: "global"}
	err = natsClient.PublishGlobal(testMsg)
	assert.Nil(t, err, "Failed to publish to global")

	// Receive message
	receivedMsg, err := sub.NextMsg(2 * time.Second)
	assert.Nil(t, err, "Did not receive published message in global")

	var received domain.ChatMessage
	err = json.Unmarshal(receivedMsg.Data, &received)
	assert.Nil(t, err, "Failed to unmarshal received message")
	assert.Equal(t, messageContent, received.Content, "Published message to global should match")
}

func TestPublishRoom(t *testing.T) {
	natsClient := setupNATSClient(t)
	defer natsClient.Close()

	room := "test_publish_room"
	messageContent := "Test Publish Room"

	// Subscribe to the room
	sub, err := natsClient.Conn.SubscribeSync("chat.events." + room)
	assert.Nil(t, err, "Failed to subscribe to room")

	// Publish message to the room
	testMsg := domain.ChatMessage{Type: domain.MessageTypeChat, Sender: "room_user", Content: messageContent, Room: room}
	err = natsClient.PublishRoom(room, testMsg)
	assert.Nil(t, err, "Failed to publish message to room")

	// Receive message
	receivedMsg, err := sub.NextMsg(2 * time.Second)
	assert.Nil(t, err, "Did not receive published message in room")

	var received domain.ChatMessage
	err = json.Unmarshal(receivedMsg.Data, &received)
	assert.Nil(t, err, "Failed to unmarshal received message")
	assert.Equal(t, messageContent, received.Content, "Published message to room should match")
}
