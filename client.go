package main

import (
	"fmt"
	"github.com/petariliev/kvstore/client"
	"log"
)

func main() {
	kvClient, err := client.New()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer kvClient.Close()

	fmt.Println("Connected to server")
	if err := kvClient.RunInteractive(); err != nil {
		log.Printf("Error during interactive session: %v", err)
	}
}
