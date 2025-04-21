package main

import (
	"log"

	"github.com/petariliev/kvstore/client"
)

func main() {
	kvClient, err := client.New()
	if err != nil {
		log.Fatalf("[FATAL] Failed to create client: %v", err)
	}
	defer kvClient.Close()

	log.Println("[INFO] Connected to server")

	if err := kvClient.RunInteractive(); err != nil {
		log.Printf("[ERROR] Error during interactive session: %v", err)
	}
}
