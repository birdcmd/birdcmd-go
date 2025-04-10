package client

import (
	"net/http"
	"fmt"
	"net/url"
	"encoding/json"
	"log"
	
	"github.com/gorilla/websocket"

	"github.com/birdcmd/birdcmd-go/pkg/config/flags"
)

func connectWebSocket() (*websocket.Conn, error) {
	u := url.URL{Scheme: flags.WsScheme, Host: flags.HostServer, Path: "/cable"}
	headers := http.Header{}
	headers.Add("Authorization", fmt.Sprintf("Bearer %s", flags.BearerToken))

	log.Printf("Connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), headers)

	return c, err
}

func subscribeToChannel(c *websocket.Conn) error {
	identifier, _ := json.Marshal(Identifier{Channel: "CommandChannel", Tunnel: flags.TunnelId})
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
