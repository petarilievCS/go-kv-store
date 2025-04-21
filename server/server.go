package server

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"

	"github.com/petariliev/kvstore/kvstore"
)

const (
	PutOK      = "OK"
	GetCommand = "GET"
	SetCommand = "SET"
	Port       = ":8080"
)

// Errors
const (
	InvalidCommand    = "ERROR: Invalid command"
	InvalidSetCommand = "ERROR: Invalid SET command"
	InvalidGetCommand = "ERROR: Invalid GET command"
	UknownCommand     = "ERROR: Uknown command"
)

var kv = kvstore.New()

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("[INFO] Client disconnected:", getAddress(conn))
				return
			}
			log.Printf("[ERROR] Unable to read from %s: %v\n", getAddress(conn), err)
			return
		}

		message = strings.TrimSpace(message)
		tokens := strings.Split(message, " ")

		response := processCommand(tokens)
		response += "\n"

		_, err = conn.Write([]byte(response))
		if err != nil {
			log.Printf("[ERROR] Error writing to %s: %v\n", getAddress(conn), err)
			return
		}
	}
}

func processCommand(tokens []string) string {
	if len(tokens) == 0 {
		log.Println("[WARN] Received empty command")
		return InvalidCommand
	}

	command := tokens[0]
	switch command {
	case GetCommand:
		if len(tokens) != 2 {
			log.Println("[WARN] Invalid GET command format")
			return InvalidGetCommand
		}
		key := tokens[1]
		value, err := kv.Get(key)
		if err != nil {
			log.Printf("[WARN] GET %s -> key not found\n", key)
			return err.Error()
		}
		log.Printf("[INFO] GET %s -> %s\n", key, value)
		return value
	case SetCommand:
		if len(tokens) != 3 {
			log.Println("[WARN] Invalid SET command format")
			return InvalidSetCommand
		}
		key, value := tokens[1], tokens[2]
		kv.Set(key, value)
		log.Printf("[INFO] SET %s %s -> OK\n", key, value)
		return PutOK
	default:
		log.Printf("[WARN] Unknown command: %s\n", command)
		return UknownCommand
	}
}

func getAddress(conn net.Conn) string {
	return conn.RemoteAddr().String()
}

func StartServer() {
	ln, err := net.Listen("tcp", Port)
	if err != nil {
		log.Fatalf("[FATAL] Failed to start server: %v\n", err)
		return
	}
	defer ln.Close()
	log.Println("[INFO] Server is listening on port 8080...")

	// Main loop
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[ERROR] Error accepting connection: %v\n", err)
			continue
		}
		log.Println("[INFO] Client connected:", getAddress(conn))
		go handleConnection(conn)
	}
}
