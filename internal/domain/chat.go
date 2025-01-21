package domain

type MessageType string

const (
	MessageTypeChat         MessageType = "chat_message"
	MessageTypeList         MessageType = "list_users"
	MessageTypeListResponse MessageType = "list_users_response"

	MessageTypeJoin          MessageType = "join_room"
	MessageTypeLeave         MessageType = "leave_room"
	MessageTypeRooms         MessageType = "list_rooms"
	MessageTypeRoomsResponse MessageType = "list_rooms_response"
)

type ChatMessage struct {
	Type      MessageType `json:"type"`
	Sender    string      `json:"sender,omitempty"`
	Content   string      `json:"content,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
	Room      string      `json:"room,omitempty"`
}
