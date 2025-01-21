package port

import "github.com/SphrGhfri/chatroom_golang_nats/internal/domain"

type ChatService interface {
	PublishMessage(msg domain.ChatMessage) error
	AddActiveUser(username string) error
	RemoveActiveUser(username string) error
	ListActiveUsers() ([]string, error)

	JoinRoom(roomName, username string) error
	LeaveRoom(roomName, username string) error
	ListRoomMembers(roomName string) ([]string, error)
	ListAllRooms() ([]string, error)
	SwitchRoom(oldRoom, newRoom, username string) error
}
