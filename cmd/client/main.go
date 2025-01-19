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
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

type ChatMessage struct {
	Sender    string `json:"sender"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

func main() {

	flag.Parse()

	username := getUsername()

	conn := connectWebSocket(username)
	defer conn.Close()

	// OS interrupt signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Start goroutine to listen for incoming messages
	done := make(chan struct{})
	go readMessages(conn, done)

	fmt.Println("Write Messages (Press Enter to Send):")
	writeMessages(conn, username, interrupt, done)
}

func getUsername() string {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter your username: ")
	scanner.Scan()
	return scanner.Text()
}

func connectWebSocket(username string) *websocket.Conn {
	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws", RawQuery: "username=" + url.QueryEscape(username)}
	log.Printf("Connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	log.Println("Connected to WebSocket server.")
	return conn
}

func readMessages(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			return
		}

		var chatMessage ChatMessage
		err = json.Unmarshal(message, &chatMessage)
		if err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		fmt.Printf("\n[%s] %s: %s\n", chatMessage.Timestamp, chatMessage.Sender, chatMessage.Content)
	}
}

func writeMessages(conn *websocket.Conn, username string, interrupt chan os.Signal, done chan struct{}) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("Interrupt received, closing connection...")
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("Error during close: %v", err)
			}
			return
		default:
			// fmt.Print("Enter message: ")
			if scanner.Scan() {
				content := scanner.Text()

				if content == "" {
					continue
				}

				message := ChatMessage{
					Sender:    username,
					Content:   content,
					Timestamp: time.Now().Format("2006-01-02 15:04:05"),
				}

				err := conn.WriteJSON(message)
				if err != nil {
					log.Printf("Error sending message: %v", err)
					return
				}

				fmt.Printf("[Sent] %s\n", content)
			}
		}
	}
}
