package client

import (
	"log"
	"time"
	"math/rand"
	"github.com/birdcmd/birdcmd-go/pkg/config/flags"
)

func StartApp() {
	for {
		c, err := connectWebSocket()
		if err != nil {
			log.Printf("Connection error: %v.\nReconnecting in 5 seconds...", err)
			time.Sleep(flags.ReconnectInterval)
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
