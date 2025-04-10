package flags

import (
	"flag"
	"time"
	"strings"
	"os"
	"log"
	"fmt"
	"github.com/birdcmd/birdcmd-go/pkg/util/version"
)

var (
	HeartbeatInterval time.Duration
	ReconnectInterval time.Duration
	BearerToken string
	TunnelId string
	IsDevMode bool
	UseCnServer bool
	WsScheme string
	HostServer string
	WsOrigin string
)

func ParseAndSetFlags() {
	c := flag.String("c", "", "(Required) Input in the format token:tunnelId")
	d := flag.Bool("d", false, "Enable development mode")
	cn := flag.Bool("cn", false, "Use China mainland server")
	showVersion := flag.Bool("v", false, "Show Version")

	flag.Usage = func() {
		fmt.Println("Usage:")
		fmt.Println("  -c string (Required) The config of birdcmd in the format token:tunnelId (separated by a colon), such as `birdcmd -c 12a7W55y(your token):ffe9-eew3(your tunnelId)`")
		fmt.Println("  -v        Show Version")
		// fmt.Println("  -d        Enable development mode (optional)")
		// fmt.Println("  -cn       Use China mainland server (developer experimental)")
		os.Exit(1)
	}

	flag.Parse()

	if *showVersion {
		fmt.Println(version.Full())
		os.Exit(0)
	}

	// Ensure -c flag is provided
	if *c == "" {
		fmt.Println("Error: -c flag is required")
		flag.Usage()
	}

	parts := strings.SplitN(*c, ":", 2)
	if len(parts) == 2 {
		BearerToken = parts[0]
		TunnelId = parts[1]
	} else {
		log.Println("Error: -c flag must be in format token:tunnelId")
		os.Exit(1)
	}

	IsDevMode = *d
	UseCnServer = *cn

	if IsDevMode {
		HeartbeatInterval = time.Duration(25) * time.Second
		ReconnectInterval = time.Duration(2) * time.Second
		WsScheme = "ws"
		HostServer = "localhost:3000"
	} else {
		HeartbeatInterval = time.Duration(45) * time.Second
		ReconnectInterval = time.Duration(10) * time.Second
		if UseCnServer {
			HostServer = "bird.gfgf.work"
		} else {
			HostServer = "www.birdcmd.com"
		}
		WsOrigin = fmt.Sprintf("https://%s", HostServer)
		WsScheme = "wss"
	}
}