package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

const (
	ServerAddress = ":8080"
	QuitCommand   = "QUIT"
)

func connectToServer() (net.Conn, error) {
	return net.Dial("tcp", ServerAddress)
}

func main() {
	conn, err := connectToServer()
	if err != nil {
		fmt.Println("Failed to connect to server:", err)
		return
	}
	defer conn.Close()
	fmt.Println("Connected to server.")

	serverReader := bufio.NewReader(conn)
	stdinReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter command: ")
		input, err := stdinReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		_, err = conn.Write([]byte(input))
		if err != nil {
			fmt.Println("Error sending command:", err)
		}

		response, err := serverReader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading message:", err)
			continue
		}
		fmt.Print(response)
	}
}
