package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

type Subscription struct {
	Command    string `json:"command"`
	Identifier string `json:"identifier"`
}

type Identifier struct {
	Channel string `json:"channel"`
	Tunnel  string `json:"tunnel"`
}

type Message struct {
	Command    string `json:"command"`
	Identifier string `json:"identifier"`
	Data       string `json:"data"`
}

const (
	heartbeatInterval = 10 * time.Second
	tunnelId          = "2f5a6361-20df-462a-805f-15ae2d880ae3"
	bearerToken       = "Bearer 12a7W55yRGKmjSvo6zmyK2P6cHkoB1kMipS8"
)

func executeCommand(command string) {
	// Set a timeout to avoid long-running/hanging commands
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		log.Printf("Error executing command: %v\nOutput: %s", err, strings.TrimSpace(output.String()))
		return
	}

	// Log successful command output
	log.Printf("\n%s", strings.TrimSpace(output.String()))
}

func handlePing() {
	return
}

func handleWelcome() {
	return
}

func handleRejectSubscription() {
	log.Printf("Connection to tunnel is refused. (Notice: check your token/uuid.)")
}

func handleConfirmSubscription(msg map[string]interface{}) {
	if identifier, ok := msg["identifier"].(string); ok {
		var id Identifier
		if err := json.Unmarshal([]byte(identifier), &id); err == nil {
			log.Printf("Connected to tunnel: %s", id.Tunnel)
		}
	}
}

func handleDisconnect(msg map[string]interface{}) string {
	if reason, ok := msg["reason"].(string); ok {
		switch reason {
		case "unauthorized":
			log.Println("Unauthorized.")
		case "invalid_request":
			log.Println("Invalid Request")
		case "server_restart":
			log.Println("Server is about to restart.")
		case "remote":
			log.Println("Remote server closed connection.")
		}
	}

	if reconnect, ok := msg["reconnect"].(bool); ok {
		if reconnect {
			return "reconnect"
		} else {
			log.Println("Connection will be closed.")
		}
	}
	return "disconnect"
}

func handleMessage(msg map[string]interface{}) {
	identifierStr, ok := msg["identifier"].(string)
	if !ok {
		log.Println("Error: Missing or invalid 'identifier' field. Full message:", msg)
		return
	}

	var id Identifier
	if err := json.Unmarshal([]byte(identifierStr), &id); err != nil {
		log.Println("Error: Failed to parse 'identifier' JSON. Full message:", msg, "Error:", err)
		return
	}

	if id.Channel != "CommandChannel" {
		log.Println("Error: Unexpected channel. Expected 'CommandChannel', got:", id.Channel)
		return
	}

	message, ok := msg["message"].(map[string]interface{})
	if !ok {
		log.Println("Error: Missing or invalid 'message' field. Full message:", msg)
		return
	}

	cmdRaw, exists := message["command"]
	if !exists {
		log.Println("Error: Missing 'command' field in 'message'. Full message:", msg)
		return
	}

	cmd, ok := cmdRaw.(string)
	if !ok {
		log.Println("Error: 'command' is not a string. Full message:", msg)
		return
	}
	executeCommand(cmd)
}

func handleWsMessage(c *websocket.Conn, msg []byte) string {
	var genericMsg map[string]interface{}
	action := ""

	if err := json.Unmarshal(msg, &genericMsg); err != nil {
		log.Println("Received message:", string(msg), "Failed to parse message:", err)
		return action
	}

	msgType, ok := genericMsg["type"].(string)
	if ok {
		switch msgType {
		case "ping":
			handlePing()
		case "welcome":
			handleWelcome()
		case "disconnect":
			action = handleDisconnect(genericMsg)
		case "confirm_subscription":
			handleConfirmSubscription(genericMsg)
		case "reject_subscription":
			handleRejectSubscription()
			action = "disconnect"
		default:
			log.Println("Unknown message type, full message: ", string(msg))
		}
		return action
	}

	handleMessage(genericMsg)
	return action
}

func connectWebSocket() (*websocket.Conn, error) {
	u := url.URL{Scheme: "ws", Host: "localhost:3000", Path: "/cable"}
	headers := http.Header{}
	headers.Add("Origin", "http://localhost:3000")
	headers.Add("Authorization", bearerToken)

	log.Printf("Connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), headers)

	return c, err
}

func subscribeToChannel(c *websocket.Conn) error {
	identifier, _ := json.Marshal(Identifier{Channel: "CommandChannel", Tunnel: tunnelId})
	subscribe := Subscription{Command: "subscribe", Identifier: string(identifier)}
	subMsg, _ := json.Marshal(subscribe)

	err := c.WriteMessage(websocket.TextMessage, subMsg)
	if err != nil {
		log.Fatalf("Subscription error: %v", err)
		return err
	}
	log.Println("Connecting to Command channel.")
	return nil
}

func listenForMessages(c *websocket.Conn, msgCh chan string, exitSig *bool) {
	for {
		_, message, err := c.ReadMessage()
		*exitSig = true
		if err != nil {
			log.Println("Message read error:", err)
			msgCh <- "reconnect"
			return
		}

		action := handleWsMessage(c, message)

		if action == "" {
			*exitSig = false
			continue
		} else if action == "reconnect" {
			log.Println("Reconnecting in a few seconds...")
			time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
			msgCh <- "reconnect"
			return
		} else if action == "disconnect" {
			log.Println("Will disconnected from websocket.")
			msgCh <- "disconnect"
			return
		} else {
			log.Println("Unknown action:", action)
			msgCh <- "disconnect"
			return
		}
	}
}

func sendHeartbeats(c *websocket.Conn, exitSig *bool) {
	time.Sleep(3 * time.Second)
	identifier, _ := json.Marshal(Identifier{Channel: "CommandChannel", Tunnel: tunnelId})
	heartbeat := Message{
		Command:    "message",
		Identifier: string(identifier),
		Data:       `{"action": "heartbeat_ping"}`,
	}
	heartbeatMsg, _ := json.Marshal(heartbeat)

	for {
		if *exitSig == true {
			log.Println("Exiting heartbeat because exit signal is true.")
			return
		}
		if err := c.WriteMessage(websocket.TextMessage, heartbeatMsg); err != nil {
			log.Println("Heartbeat send error:", err, "Stopping heartbeats.")
			return
		}
		log.Println("Sent heartbeat ping.")
		time.Sleep(5 * time.Second)
	}
}

func main() {
	for {
		c, err := connectWebSocket()
		if err != nil {
			log.Printf("Connection error: %v.\nReconnecting in 5 seconds...", err)
			time.Sleep(time.Duration(5) * time.Second)
			continue
		}

		err = subscribeToChannel(c)
		if err != nil {
			c.Close()
			return
		}

		msgCh := make(chan string)
		exitSig := false

		go listenForMessages(c, msgCh, &exitSig)
		go sendHeartbeats(c, &exitSig)

		select {
		case action := <-msgCh:
			c.Close()
			close(msgCh)
			if action == "reconnect" {
				log.Println("Reconnecting in a few seconds...")
				continue
			}
			if action == "disconnect" {
				log.Println("Disconnected from websocket.")
				return
			}
		}
	}
	log.Println("Exited.")
}
