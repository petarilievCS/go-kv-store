package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Println("Connected to server.")

	serverOutput := bufio.NewReader(conn)
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter command: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input")
		}

		conn.Write([]byte(input))
		serverResponse, err := serverOutput.ReadString('\n')
		if err != nil {
			fmt.Print("Server Error:", err)
		} else {
			fmt.Print(serverResponse)
		}

	}
}
