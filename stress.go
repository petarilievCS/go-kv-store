package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	serverAddress = ":8080"
	numClients    = 100
)

func main() {
	var wg sync.WaitGroup

	for i := 0; i < numClients; i++ {
		wg.Add(1)

		go func(clientID int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", serverAddress)
			if err != nil {
				log.Printf("[ERROR] Error connecting to server: %s", err)
			}
			defer conn.Close()

			reader := bufio.NewReader(conn)

			key := fmt.Sprintf("key%d", clientID)
			value := fmt.Sprintf("value%d", clientID)
			setCommand := fmt.Sprintf("SET %s %s\n", key, value)
			getCommand := fmt.Sprintf("GET %s\n", key)

			_, err = conn.Write([]byte(setCommand))
			if err != nil {
				log.Printf("[ERROR] Error sending command: %s", err)
			}

			response, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("[ERROR] Error receiving response: %s", err)
			}

			response = strings.TrimSpace(response)
			if response != "OK" {
				log.Printf("[MISMATCH] Client %d: expected %s, got %s", clientID, "OK", response)
			}

			time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

			_, err = conn.Write([]byte(getCommand))
			if err != nil {
				log.Printf("[ERROR] Error sending command: %s", err)
			}

			response, err = reader.ReadString('\n')
			if err != nil {
				log.Printf("[ERROR] Error receiving response: %s", err)
			}

			response = strings.TrimSpace(response)
			if response != value {
				log.Printf("[MISMATCH] Client %d: expected %s, got %s", clientID, value, response)
			}
		}(i)
	}

	wg.Wait()

	log.Printf("[DONE] %d clients finished\n", numClients)
}
