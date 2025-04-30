package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/peterh/liner"
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
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)
	for {
		cmd, err := line.Prompt("kv> ")
		if err != nil {
			if err == liner.ErrPromptAborted {
				fmt.Println("Aborted.")
				break
			}
			log.Printf("[ERROR] Error reading input: %v", err)
			continue
		}

		cmd = strings.TrimSpace(cmd)
		if cmd == ExitCommand || cmd == QuitCommand {
			log.Println("[INFO] Client exited interactive session")
			fmt.Println("Bye 👋")
			break
		}

		line.AppendHistory(cmd)

		if err := validateInput(cmd); err != nil {
			fmt.Println(err)
			continue
		}

		response, err := c.SendCommand(cmd)
		if err != nil {
			log.Printf("[ERROR] Command failed: %v", err)
			continue
		}

		fmt.Println(response)
	}
	return nil
}

// Helpers

func validateInput(input string) error {
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return errors.New("[ERROR]: Empty input")
	}
	cmd := strings.ToUpper(tokens[0])

	switch cmd {
	case "SET":
		if len(tokens) != 3 {
			return errors.New("[ERROR] Invalid SET command. Format: SET <key> <value>")
		}
	case "GET", "DELETE":
		if len(tokens) != 2 {
			return fmt.Errorf("[ERROR] Invalid %s command. Format: %s <key>", cmd, cmd)
		}
	case "SETEX":
		if len(tokens) != 4 {
			return errors.New("[ERROR] Invalid SETEX command. Format: SETEX <key> <value> <ttl_seconds>")
		}
	case "DELETEEX":
		if len(tokens) != 3 {
			return errors.New("[ERROR] Invalid DELETEEX command. Format: DELETEEX <key> <seconds>")
		}
	case "PING", "STATS", "KEYS":
		if len(tokens) != 1 {
			return fmt.Errorf("[ERROR] Invalid %s command. Format: %s", cmd, cmd)
		}
	}
	return nil
}
