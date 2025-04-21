package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	ServerAddress = ":8080"
	QuitCommand   = "quit"
	ExitCommand   = "exit"
)

type KVClient struct {
	conn   net.Conn
	reader *bufio.Reader
}

func New() (*KVClient, error) {
	conn, err := net.Dial("tcp", ServerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}

	reader := bufio.NewReader(conn)
	client := KVClient{
		conn:   conn,
		reader: reader,
	}
	return &client, nil
}

func (c *KVClient) Close() error {
	return c.conn.Close()
}

func (c *KVClient) SendCommand(command string) (string, error) {
	_, err := c.conn.Write([]byte(command + "\n"))
	if err != nil {
		return "", fmt.Errorf("error sending command: %v", err)
	}

	response, err := c.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}
	return strings.TrimSpace(response), nil
}

func (c *KVClient) RunInteractive() error {
	stdinReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("kv> ")
		input, err := stdinReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == ExitCommand || input == QuitCommand {
			fmt.Println("Bye 👋")
			break
		}

		if input == "" {
			continue // Don't send empty commands
		}

		response, err := c.SendCommand(input)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		fmt.Println(response)
	}
	return nil
}
