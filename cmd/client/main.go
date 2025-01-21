package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

// Define your message types
const (
	MessageTypeChat         = "chat_message"
	MessageTypeList         = "list_users"
	MessageTypeListResponse = "list_users_response" // if server responds with a list
)

// ChatMessage is the typed message structure exchanged with the server
type ChatMessage struct {
	Type      string `json:"type"`              // e.g. "chat_message", "list_users", etc.
	Sender    string `json:"sender,omitempty"`  // only relevant for chat
	Content   string `json:"content,omitempty"` // message text or list of users
	Timestamp string `json:"timestamp,omitempty"`
}

func main() {
	flag.Parse()

	username := getUsername()

	conn := connectWebSocket(username)
	defer conn.Close()

	// Listen for OS interrupt signals (Ctrl+C)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Goroutine to listen for incoming WS messages
	done := make(chan struct{})
	go readMessages(conn, done)

	fmt.Println("Type '/list' to see active users, or type any message to chat:")
	writeMessages(conn, username, interrupt, done)
}

// getUsername simply asks user input for their name
func getUsername() string {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter your username: ")
	scanner.Scan()
	return scanner.Text()
}

// connectWebSocket dials the WS endpoint with the given username
func connectWebSocket(username string) *websocket.Conn {
	u := url.URL{
		Scheme:   "ws",
		Host:     *addr,
		Path:     "/ws",
		RawQuery: "username=" + url.QueryEscape(username),
	}
	log.Printf("Connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	log.Println("Connected to WebSocket server.")
	return conn
}

// readMessages continuously reads messages from the server,
// unmarshals them into ChatMessage, and prints them appropriately.
func readMessages(conn *websocket.Conn, done chan struct{}) {
	defer close(done)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			return
		}

		var msg ChatMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		switch msg.Type {
		case MessageTypeChat:
			// Regular chat message from the server or other users
			fmt.Printf("\n[%s] %s: %s\n", msg.Timestamp, msg.Sender, msg.Content)

		case MessageTypeListResponse:
			// e.g. a list of active users
			fmt.Printf("\n[System] %s\n", msg.Content)

		default:
			// Unrecognized type—just print raw content
			fmt.Printf("\n[Unknown Type: %s] %s\n", msg.Type, msg.Content)
		}
	}
}

// writeMessages reads user input from stdin. If the user types a recognized
// command (like "/list"), we send a typed message. Otherwise, we send a normal chat message.
func writeMessages(conn *websocket.Conn, username string, interrupt chan os.Signal, done chan struct{}) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("Interrupt received, closing connection...")
			err := conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("Error during close: %v", err)
			}
			return
		default:
			if scanner.Scan() {
				input := scanner.Text()

				// Skip empty lines
				if strings.TrimSpace(input) == "" {
					continue
				}

				var msg ChatMessage

				// Check if user typed a command
				if strings.HasPrefix(input, "/list") {
					// Command to list active users
					msg.Type = MessageTypeList
				} else {
					// Otherwise, treat it as a normal chat message
					msg.Type = MessageTypeChat
					msg.Sender = username
					msg.Content = input
					msg.Timestamp = time.Now().Format("2006-01-02 15:04:05")
				}

				if err := conn.WriteJSON(msg); err != nil {
					log.Printf("Error sending message: %v", err)
					return
				}

				// If it's a chat message, optionally echo what was sent
				if msg.Type == MessageTypeChat {
					fmt.Printf("[Sent] %s\n", input)
				} else {
					fmt.Printf("[Sent Command] %s\n", input)
				}
			}
		}
	}
}
