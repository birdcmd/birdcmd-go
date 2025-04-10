package main

import (
	"log"
	"github.com/birdcmd/birdcmd-go/pkg/config/flags"
	"github.com/birdcmd/birdcmd-go/client"
)

func main() {
	flags.ParseAndSetFlags()
	client.StartApp()
	log.Println("Exited.")
}
