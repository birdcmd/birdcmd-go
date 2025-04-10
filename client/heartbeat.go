package client

import (
	"encoding/json"
	"log"
	"time"
	"math/rand"

	"github.com/gorilla/websocket"
	"github.com/birdcmd/birdcmd-go/pkg/config/flags"
)

func sendHeartbeats(c *websocket.Conn, exitSig *bool) {
	identifier, _ := json.Marshal(Identifier{Channel: "CommandChannel", Tunnel: flags.TunnelId})
	heartbeat := Message{
		Command:    "message",
		Identifier: string(identifier),
		Data:       `{"action": "heartbeat_ping"}`,
	}
	heartbeatMsg, _ := json.Marshal(heartbeat)

	for {
		time.Sleep(flags.HeartbeatInterval + time.Duration(rand.Intn(10)-5) * time.Second)
		if *exitSig == true {
			log.Println("Exiting heartbeat from stale connection.")
			return
		}
		if err := c.WriteMessage(websocket.TextMessage, heartbeatMsg); err != nil {
			log.Println("Heartbeat send error:", err, "Stopping heartbeats.")
			return
		}
		if flags.IsDevMode {
			log.Println("Sent heartbeat ping.")
		}
	}
}