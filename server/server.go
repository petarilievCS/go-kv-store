package server

import (
	"bufio"
	"fmt"
	"io"
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
				return
			}
			fmt.Println("Error reading from connection:", err)
			return
		}

		message = strings.TrimSpace(message)
		tokens := strings.Split(message, " ")

		response := processCommand(tokens)
		response += "\n"

		_, err = conn.Write([]byte(response))
		if err != nil {
			fmt.Println("Error writing to connection:", err)
			return
		}
	}
}

func processCommand(tokens []string) string {
	if len(tokens) == 0 {
		return InvalidCommand
	}

	command := tokens[0]
	switch command {
	case GetCommand:
		if len(tokens) != 2 {
			return InvalidGetCommand
		}
		key := tokens[1]
		value, err := kv.Get(key)
		if err != nil {
			return err.Error()
		}
		return value
	case SetCommand:
		if len(tokens) != 3 {
			return InvalidSetCommand
		}
		key, value := tokens[1], tokens[2]
		kv.Set(key, value)
		return PutOK
	default:
		return UknownCommand
	}
}

func StartServer() {
	ln, err := net.Listen("tcp", Port)
	if err != nil {
		fmt.Println("Failed to start server:", err)
		return
	}
	defer ln.Close()
	fmt.Println("Server is listening on port 8080...")

	// Main loop
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection", err)
			continue
		}
		go handleConnection(conn)
	}
}
