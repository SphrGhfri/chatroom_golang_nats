package domain

type MessageType string

const (
	MessageTypeChat   MessageType = "chat_message"
	MessageTypeSystem MessageType = "system_message" // Add this for system messages
	MessageTypeList   MessageType = "list_users"
	MessageTypeRooms  MessageType = "list_rooms"
	MessageTypeJoin   MessageType = "join_room"
	MessageTypeLeave  MessageType = "leave_room"
)

type ChatMessage struct {
	Type      MessageType `json:"type"`
	Sender    string      `json:"sender,omitempty"`
	Content   string      `json:"content,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
	Room      string      `json:"room,omitempty"`
}
