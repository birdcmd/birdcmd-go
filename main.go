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
	"flag"
	"os"
	"fmt"
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

var (
	heartbeatInterval time.Duration
	reconnectInterval time.Duration
	bearerToken string
	tunnelId string
	isDevMode bool
	useCnServer bool
	wsScheme string
	hostServer string
	wsOrigin string
)

func executeCommand(command string) {
	// Set a timeout to avoid long-running/hanging commands
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Env = os.Environ()
	
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
	log.Printf("Connection to tunnel is refused. (Notice: check your token/tunnelId.)")
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
		infoRaw, exists := message["info"]
		if exists {
			log.Println("[Info] Server: ", infoRaw)
		}	else {
			log.Println("Error: Missing 'command' field in 'message'. Full message:", msg)
		}
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
	u := url.URL{Scheme: wsScheme, Host: hostServer, Path: "/cable"}
	headers := http.Header{}
	headers.Add("Authorization", fmt.Sprintf("Bearer %s", bearerToken))

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
			msgCh <- "reconnect"
			return
		} else if action == "disconnect" {
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
	identifier, _ := json.Marshal(Identifier{Channel: "CommandChannel", Tunnel: tunnelId})
	heartbeat := Message{
		Command:    "message",
		Identifier: string(identifier),
		Data:       `{"action": "heartbeat_ping"}`,
	}
	heartbeatMsg, _ := json.Marshal(heartbeat)

	for {
		time.Sleep(heartbeatInterval)
		if *exitSig == true {
			log.Println("Exiting heartbeat because from stale connection.")
			return
		}
		if err := c.WriteMessage(websocket.TextMessage, heartbeatMsg); err != nil {
			log.Println("Heartbeat send error:", err, "Stopping heartbeats.")
			return
		}
		if isDevMode {
			log.Println("Sent heartbeat ping.")
		}
	}
}

func startApp() {
	for {
		c, err := connectWebSocket()
		if err != nil {
			log.Printf("Connection error: %v.\nReconnecting in 5 seconds...", err)
			time.Sleep(reconnectInterval)
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
				time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
				continue
			}
			if action == "disconnect" {
				log.Println("Disconnecting.")
				return
			}
		}
	}
}

func parseAndSetFlags() {
	c := flag.String("c", "", "(Required) Input in the format token:tunnelId")
	d := flag.Bool("d", false, "Enable development mode")
	cn := flag.Bool("cn", false, "Use China mainland server")

	flag.Usage = func() {
		fmt.Println("Usage:")
		fmt.Println("  -c string (Required) Input in the format token:tunnelId (separated by a colon), such as `birdcmd -c 12a7W55y(your token):ffe9-eew3(your tunnelId)`")
		fmt.Println("  -d        Enable development mode (optional)")
		fmt.Println("  -cn       Use China mainland server (developer experimental)")
		os.Exit(1)
	}

	flag.Parse()

	// Ensure -c flag is provided
	if *c == "" {
		fmt.Println("Error: -c flag is required")
		flag.Usage()
	}

	parts := strings.SplitN(*c, ":", 2)
	if len(parts) == 2 {
		bearerToken = parts[0]
		tunnelId = parts[1]
	} else {
		log.Println("Error: -c flag must be in format token:tunnelId")
		os.Exit(1)
	}

	isDevMode = *d
	useCnServer = *cn

	if isDevMode {
		heartbeatInterval = time.Duration(5) * time.Second
		reconnectInterval = time.Duration(2) * time.Second
		wsScheme = "ws"
		hostServer = "localhost:3000"
	} else {
		heartbeatInterval = time.Duration(45) * time.Second
		reconnectInterval = time.Duration(10) * time.Second
		if useCnServer {
			hostServer = "bird.gfgf.work"
		} else {
			hostServer = "www.birdcmd.com"
		}
		wsOrigin = fmt.Sprintf("https://%s", hostServer)
		wsScheme = "wss"
	}
}

func main() {
	parseAndSetFlags()
	startApp()
	log.Println("Exited.")
}
