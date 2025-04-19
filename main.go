package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

const INVALID_COMMAND_ERROR = "Invalid command."

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	fmt.Println("Server is listening on port 8080...")

	conn, err := ln.Accept()
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Println("Client connected!")

	reader := bufio.NewReader(conn)

	// Loop for commands
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		message = string(message)
		message = strings.TrimSpace(message)
		tokens := strings.Split(message, " ")

		response := ""
		if len(tokens) == 2 {
			if tokens[0] == "GET" {
				response = message
			} else {
				response = INVALID_COMMAND_ERROR
			}
		} else if len(tokens) == 3 {
			if tokens[0] == "SET" {
				response = message
			} else {
				response = INVALID_COMMAND_ERROR
			}
		} else {
			response = INVALID_COMMAND_ERROR
		}
		response = response + "\n"
		conn.Write([]byte(response))
	}
}
