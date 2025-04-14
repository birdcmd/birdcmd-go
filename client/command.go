package client

import (
	"bytes"
	"os/exec"
	"strings"
	"time"
	"os"
	"encoding/json"
	"log"

	"github.com/birdcmd/birdcmd-go/pkg/config/flags"
)

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


func executeCommand(command string) {
	// Set a timeout to avoid long-running/hanging commands
	timeoutSec := 10*time.Second
	if flags.EnableLongRunning {
		timeoutSec = 600*time.Second
	}

	log.Println("Running:", command)

	cmd := exec.Command("sh", "-c", command)
	cmd.Env = os.Environ()
	
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	
	// Start the process
	if err := cmd.Start(); err != nil {
			log.Printf("Error starting command: %v", err)
			return
	}

	// Set up a channel for completion
	done := make(chan error, 1)
	go func() {
			done <- cmd.Wait()
	}()
	
	// Wait for completion or timeout
	select {
	case err := <-done:
			if err != nil {
					log.Printf("Error executing command: %v\nOutput: %s", err, strings.TrimSpace(output.String()))
					return
			}
	case <-time.After(timeoutSec):
			// Force kill the process
			if err := cmd.Process.Kill(); err != nil {
					log.Printf("Error killing process: %v", err)
			}
			log.Printf("Command timed out after %s seconds\nPartial output: %s", timeoutSec, strings.TrimSpace(output.String()))
			return
	}
	log.Printf("\n%s", strings.TrimSpace(output.String()))
}
