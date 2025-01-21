package websocket

import (
	"fmt"
	"strings"
	"time"

	"github.com/SphrGhfri/chatroom_golang_nats/internal/domain"
	"github.com/SphrGhfri/chatroom_golang_nats/internal/port"
	"github.com/SphrGhfri/chatroom_golang_nats/pkg/logger"
	gws "github.com/gorilla/websocket"
)

type Connection struct {
	Ws          *gws.Conn
	Send        chan interface{}
	Hub         *Hub
	Username    string
	ChatService port.ChatService
	Logger      logger.Logger

	CurrentRoom string // "global" by default, or a custom room
}

func (c *Connection) ReadPump() {
	defer func() {
		if c.CurrentRoom != "" {
			if err := c.ChatService.LeaveRoom(c.CurrentRoom, c.Username); err != nil {
				c.Logger.Errorf("LeaveRoom error: %v", err)
			}
		}
		if err := c.ChatService.RemoveActiveUser(c.Username); err != nil {
			c.Logger.Errorf("RemoveActiveUser error: %v", err)
		}
		c.Hub.Unregister <- c
		c.Ws.Close()
	}()

	for {
		var msg domain.ChatMessage
		if err := c.Ws.ReadJSON(&msg); err != nil {
			c.Logger.Errorf("[CONNECTION] ReadJSON error: %v", err)
			break
		}

		switch msg.Type {

		case domain.MessageTypeList:
			// /list or /list room
			room := strings.TrimSpace(msg.Room)
			if room == "" {
				users, err := c.ChatService.ListActiveUsers()
				if err != nil {
					c.Logger.Errorf("List users error: %v", err)
					continue
				}
				c.sendSystemMsg("All active users: "+join(users, ", "), now())
			} else {
				members, err := c.ChatService.ListRoomMembers(room)
				if err != nil {
					c.Logger.Errorf("ListRoomMembers error: %v", err)
					continue
				}
				c.sendSystemMsg("Users in "+room+": "+join(members, ", "), now())
			}

		case domain.MessageTypeRooms:
			rooms, err := c.ChatService.ListAllRooms()
			if err != nil {
				c.Logger.Errorf("ListAllRooms error: %v", err)
				continue
			}
			c.sendSystemMsg("Available rooms: "+join(rooms, ", "), now())

		case domain.MessageTypeJoin:
			oldRoom := c.CurrentRoom
			newRoom := strings.TrimSpace(msg.Room)
			if newRoom == "" {
				continue
			}

			if err := c.ChatService.SwitchRoom(oldRoom, newRoom, c.Username); err != nil {
				c.Logger.Errorf("SwitchRoom error: %v", err)
				continue
			}
			c.CurrentRoom = newRoom

			joinMsg := domain.ChatMessage{
				Type:      domain.MessageTypeChat,
				Sender:    "System",
				Content:   fmt.Sprintf("%s joined room %s", c.Username, newRoom),
				Timestamp: now(),
				Room:      newRoom,
			}
			if err := c.ChatService.PublishMessage(joinMsg); err != nil {
				c.Logger.Errorf("PublishMessage error: %v", err)
				continue
			}

		case domain.MessageTypeLeave:
			oldRoom := c.CurrentRoom
			if oldRoom == "" {
				oldRoom = "global"
			}
			if err := c.ChatService.SwitchRoom(oldRoom, "global", c.Username); err != nil {
				c.Logger.Errorf("SwitchRoom error: %v", err)
				continue
			}
			c.CurrentRoom = "global"

			leftMsg := domain.ChatMessage{
				Type:      domain.MessageTypeChat,
				Sender:    "System",
				Content:   c.Username + " left room " + oldRoom,
				Timestamp: now(),
				Room:      oldRoom,
			}
			joinMsg := domain.ChatMessage{
				Type:      domain.MessageTypeChat,
				Sender:    "System",
				Content:   c.Username + " joined room global",
				Timestamp: now(),
				Room:      "global",
			}
			if err := c.ChatService.PublishMessage(leftMsg); err != nil {
				c.Logger.Errorf("PublishMessage error: %v", err)
				continue
			}
			if err := c.ChatService.PublishMessage(joinMsg); err != nil {
				c.Logger.Errorf("PublishMessage error: %v", err)
				continue
			}

		case domain.MessageTypeChat:
			if msg.Room == "" {
				msg.Room = c.CurrentRoom
				if msg.Room == "" {
					msg.Room = "global"
				}
			}
			msg.Sender = c.Username
			msg.Timestamp = now()

			// Publish message to the appropriate NATS room
			if err := c.ChatService.PublishMessage(msg); err != nil {
				c.Logger.Errorf("PublishMessage error: %v", err)
			}

		default:
			c.Logger.Infof("[CONNECTION] Unknown type: %s", msg.Type)
		}
	}
}

func (c *Connection) WritePump() {
	defer c.Ws.Close()

	for msg := range c.Send {
		if err := c.Ws.WriteJSON(msg); err != nil {
			c.Logger.Errorf("[CONNECTION] WriteJSON error: %v", err)
			break
		}
	}
}

// Helpers
func (c *Connection) sendSystemMsg(content, timestamp string) {
	reply := domain.ChatMessage{
		Type:      domain.MessageTypeListResponse,
		Sender:    "System",
		Content:   content,
		Timestamp: timestamp,
	}
	c.Send <- reply
}

func join(strs []string, sep string) string {
	return strings.Join(strs, sep)
}
func now() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
