package client

import (
	"bufio"
	"fmt"
	"io"
	"log"
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

	var response strings.Builder
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return "", fmt.Errorf("server disconnected")
			}
			return "", fmt.Errorf("[ERROR] Reading response: %v", err)
		}
		if strings.TrimSpace(line) == "END" {
			break
		}
		response.WriteString(line)
	}
	return strings.TrimSpace(response.String()), nil
}

func (c *KVClient) RunInteractive() error {
	stdinReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("kv> ") // keep this for user interaction
		input, err := stdinReader.ReadString('\n')
		if err != nil {
			log.Printf("[ERROR] Error reading input: %v", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == ExitCommand || input == QuitCommand {
			log.Println("[INFO] Client exited interactive session")
			fmt.Println("Bye 👋")
			break
		}

		if input == "" {
			continue
		}

		response, err := c.SendCommand(input)
		if err != nil {
			log.Printf("[ERROR] Command failed: %v", err)
			continue
		}

		fmt.Println(response)
	}
	return nil
}
