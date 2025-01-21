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

// Matching server domain
const (
	MessageTypeChat          = "chat_message"
	MessageTypeUsers         = "list_users"
	MessageTypeUsersResponse = "list_users_response"
	MessageTypeRooms         = "list_rooms"
	MessageTypeRoomsResponse = "list_rooms_response"
	MessageTypeJoin          = "join_room"
	MessageTypeLeave         = "leave_room"

	MessageTypeUserExists = "username_exists"
)

type ChatMessage struct {
	Type      string `json:"type"`
	Sender    string `json:"sender,omitempty"`
	Content   string `json:"content,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Room      string `json:"room,omitempty"`
}

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {
	flag.Parse()

	username := promptUsername()
	conn := connectWebSocket(username)
	defer conn.Close()

	// OS interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})
	go readMessages(conn, done)

	printHelp()

	currentRoom := "global"
	writeMessages(conn, username, &currentRoom, interrupt, done)
}

func promptUsername() string {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter your username: ")
	scanner.Scan()
	return scanner.Text()
}

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
		log.Fatalf("Failed to connect: %v", err)
	}
	log.Println("Connected.")
	return conn
}

func readMessages(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			return
		}
		var msg ChatMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("JSON parse error: %v", err)
			continue
		}

		switch msg.Type {
		case MessageTypeChat:
			fmt.Printf("\n[%s][%s] %s\n", msg.Timestamp, msg.Sender, msg.Content)

		case MessageTypeUsersResponse:
			fmt.Printf("\n[System] %s\n", msg.Content)

		case MessageTypeRoomsResponse:
			fmt.Printf("\n[System] %s\n", msg.Content)

		case MessageTypeUserExists:
			fmt.Printf("\n[System] %s\n", msg.Content)

		default:
			fmt.Printf("\n[Unknown] Type=%s Content=%s\n", msg.Type, msg.Content)
		}
	}
}

func writeMessages(conn *websocket.Conn, username string, currentRoom *string, interrupt chan os.Signal, done chan struct{}) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("Interrupt received, closing connection...")
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		default:
			if scanner.Scan() {
				input := scanner.Text()
				if strings.TrimSpace(input) == "" {
					continue
				}

				var msg ChatMessage
				fields := strings.Fields(input)

				switch {
				case strings.HasPrefix(input, "/users"):
					msg.Type = MessageTypeUsers
					if len(fields) == 2 {
						msg.Room = fields[1]
					}

				case strings.HasPrefix(input, "/rooms"):
					msg.Type = MessageTypeRooms

				case strings.HasPrefix(input, "/join"):
					if len(fields) < 2 {
						fmt.Println("Usage: /join <roomName>")
						continue
					}
					msg.Type = MessageTypeJoin
					msg.Room = fields[1]
					*currentRoom = fields[1]

				case strings.HasPrefix(input, "/leave"):
					msg.Type = MessageTypeLeave
					msg.Room = *currentRoom
					*currentRoom = "global"

				case strings.HasPrefix(input, "/help"):
					printHelp()

				default:
					// Normal chat
					msg.Type = MessageTypeChat
					msg.Sender = username
					msg.Content = input
					msg.Timestamp = time.Now().Format("2006-01-02 15:04:05")
					msg.Room = *currentRoom
				}

				if err := conn.WriteJSON(msg); err != nil {
					log.Printf("WriteJSON error: %v", err)
					return
				}

				// optional local echo
				if msg.Type == MessageTypeChat {
					if msg.Room == "global" {
						fmt.Printf("[Sent to GLOBAL] %s\n", input)
					} else {
						fmt.Printf("[Sent to %s] %s\n", msg.Room, input)
					}
				} else {
					fmt.Printf("[Sent Command] %s\n", input)
				}
			}
		}
	}
}

func printHelp() {
	fmt.Println(`Commands:
    /help
    /users           -> list all active users
    /users <room>    -> list all users in <room>
    /rooms           -> list all active rooms
    /join <room>     -> switch from your current room to <room>
    /leave           -> go back to "global" room
    Any other text   -> send a normal chat to your current room (default is "global").`)
}
