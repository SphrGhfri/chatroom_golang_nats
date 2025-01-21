package domain

type MessageType string

const (
	MessageTypeChat MessageType = "chat_message"
	MessageTypeList MessageType = "list_users"
)

type ChatMessage struct {
	Type      MessageType `json:"type"`
	Sender    string      `json:"sender,omitempty"`
	Content   string      `json:"content,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
}
