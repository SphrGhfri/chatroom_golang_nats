package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MessageType represents different types of messages that can be exchanged
type MessageType string

// Message types for client-server communication
const (
	MessageTypeChat          MessageType = "chat_message"
	MessageTypeUsers         MessageType = "list_users"
	MessageTypeUsersResponse MessageType = "list_users_response"
	MessageTypeRooms         MessageType = "list_rooms"
	MessageTypeRoomsResponse MessageType = "list_rooms_response"
	MessageTypeJoin          MessageType = "join_room"
	MessageTypeLeave         MessageType = "leave_room"
	MessageTypeUserExists    MessageType = "username_exists"
)

// ChatMessage represents the structure of messages exchanged between client and server
type ChatMessage struct {
	Type      string `json:"type"`
	Sender    string `json:"sender,omitempty"`
	Content   string `json:"content,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Room      string `json:"room,omitempty"`
}

// Client represents a chat client instance with its connection and state
type Client struct {
	conn        *websocket.Conn
	username    string
	currentRoom string
	done        chan struct{}
	mutex       sync.Mutex
}

// NewClient creates and initializes a new chat client
func NewClient(conn *websocket.Conn, username string) *Client {
	return &Client{
		conn:        conn,
		username:    username,
		currentRoom: "global",
		done:        make(chan struct{}),
	}
}

// setCurrentRoom safely updates the client's current room
func (c *Client) setCurrentRoom(room string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.currentRoom = room
}

// main initializes and runs the chat client
func main() {
	// Setup command line flags
	addr := flag.String("addr", "localhost:8080", "server address")
	flag.Parse()

	// Initialize client connection
	username := promptUsername()
	conn := connectWebSocket(*addr, username)
	if conn == nil {
		os.Exit(1)
	}
	defer conn.Close()

	// Create and start client
	client := NewClient(conn, username)
	printHelp()

	// Start message reader in background
	go client.readMessages()

	// Handle user input until program exits
	client.handleInput()
}

// promptUsername asks the user to input their username
func promptUsername() string {
	fmt.Print("Enter your username: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

// connectWebSocket establishes a WebSocket connection with the chat server
func connectWebSocket(addr, username string) *websocket.Conn {
	u := url.URL{
		Scheme:   "ws",
		Host:     addr,
		Path:     "/ws",
		RawQuery: "username=" + url.QueryEscape(username),
	}
	log.Printf("Connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("Failed to connect: %v", err)
		return nil
	}
	log.Println("Connected successfully")
	return conn
}

// readMessages continuously reads incoming messages from the WebSocket connection
func (c *Client) readMessages() {
	defer close(c.done)
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Printf("Read error: %v", err)
			}
			return
		}

		var msg ChatMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("JSON parse error: %v", err)
			continue
		}

		c.displayMessage(msg)
	}
}

// displayMessage formats and displays received messages to the user
func (c *Client) displayMessage(msg ChatMessage) {
	switch MessageType(msg.Type) {
	case MessageTypeChat:
		fmt.Printf("\n[%s][%s] %s\n", msg.Timestamp, msg.Sender, msg.Content)
	case MessageTypeUsersResponse, MessageTypeRoomsResponse, MessageTypeUserExists:
		fmt.Printf("\n[System] %s\n", msg.Content)
	default:
		fmt.Printf("\n[Info] %s\n", msg.Content)
	}
	fmt.Print("> ")
}

// handleInput processes user input and handles command execution
func (c *Client) handleInput() {
	scanner := bufio.NewScanner(os.Stdin)
	for {

		fmt.Print("> ")
		if !scanner.Scan() {
			return
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if err := c.processCommand(input); err != nil {
			log.Printf("Error: %v", err)
		}
	}

}

// processCommand determines if input is a command or chat message and routes accordingly
func (c *Client) processCommand(input string) error {
	if len(input) == 0 {
		return nil
	}

	if input[0] == '/' {
		return c.handleCommand(strings.Fields(input))
	}

	return c.sendChatMessage(input)
}

// handleCommand processes special commands starting with '/'
func (c *Client) handleCommand(fields []string) error {
	cmd := fields[0]
	switch cmd {
	case "/help":
		printHelp()
		return nil

	case "/users":
		msg := ChatMessage{Type: string(MessageTypeUsers)}
		if len(fields) == 2 {
			msg.Room = fields[1]
		}
		return c.conn.WriteJSON(msg)

	case "/rooms":
		return c.conn.WriteJSON(ChatMessage{Type: string(MessageTypeRooms)})

	case "/join":
		if len(fields) < 2 {
			return fmt.Errorf("usage: /join <roomName>")
		}
		c.setCurrentRoom(fields[1])
		return c.conn.WriteJSON(ChatMessage{
			Type: string(MessageTypeJoin),
			Room: fields[1],
		})

	case "/leave":
		msg := ChatMessage{
			Type: string(MessageTypeLeave),
			Room: c.currentRoom,
		}
		c.setCurrentRoom("global")
		return c.conn.WriteJSON(msg)

	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

// sendChatMessage sends a regular chat message to the current room
func (c *Client) sendChatMessage(content string) error {
	msg := ChatMessage{
		Type:      string(MessageTypeChat),
		Sender:    c.username,
		Content:   content,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Room:      c.currentRoom,
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	if c.currentRoom == "global" {
		fmt.Printf("[Sent to GLOBAL] %s\n", content)
	} else {
		fmt.Printf("[Sent to %s] %s\n", c.currentRoom, content)
	}
	return nil
}

// printHelp displays available commands and their usage
func printHelp() {
	fmt.Print(`
Available Commands:
    /help           -> show this help message
    /users          -> list all active users
    /users <room>   -> list users in specific room
    /rooms          -> list all active rooms
    /join <room>    -> join or switch to a room
    /leave          -> leave current room (returns to global)
    
Just type your message to chat in the current room
Current room is shown in your message confirmations
`)
}
