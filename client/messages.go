package client

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
)

func listenForMessages(c *websocket.Conn, msgCh chan string, exitSig *bool) {
	for {
		_, message, err := c.ReadMessage()

		if err != nil {
			*exitSig = true
			log.Println("Message read error:", err)
			msgCh <- "reconnect"
			return
		}

		action := handleWsMessage(c, message)

		if action == "" {
			*exitSig = false
			continue
		} else if action == "reconnect" {
			*exitSig = true
			msgCh <- "reconnect"
			return
		} else if action == "disconnect" {
			*exitSig = true
			msgCh <- "disconnect"
			return
		} else {
			*exitSig = true
			log.Println("Unknown action:", action)
			msgCh <- "disconnect"
			return
		}
	}
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

func handlePing() {
	return
}

func handleWelcome() {
	return
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
