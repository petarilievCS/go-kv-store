package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

const (
	InvalidCommandError = "(invalid command)"
	GetCommand          = "GET"
	SetCommand          = "SET"
	Port                = ":8080"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
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
	switch len(tokens) {
	case 2:
		if tokens[0] == GetCommand {
			return strings.Join(tokens, " ")
		}
	case 3:
		if tokens[0] == SetCommand {
			return strings.Join(tokens, " ")
		}
	}
	return InvalidCommandError
}

func main() {
	ln, err := net.Listen("tcp", Port)
	if err != nil {
		fmt.Println("Failed to start server:", err)
		return
	}
	defer ln.Close()
	fmt.Println("Server is listening on port 8080...")

	conn, err := ln.Accept()
	if err != nil {
		fmt.Println("Error accepting connection", err)
	}
	handleConnection(conn)
}
