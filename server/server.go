package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/petariliev/kvstore/kvstore"
)

const (
	PutOK        = "OK"
	GetCommand   = "GET"
	SetCommand   = "SET"
	SetexCommand = "SETEX"
	StatsCommand = "STATS"
	Port         = ":8080"
	Timeout      = 30
)

// Errors
const (
	InvalidCommand      = "ERROR: Invalid command. Known commands: SET, GET, SETEX"
	InvalidSetCommand   = "ERROR: Invalid SET command. Format: SET <key> <value>"
	InvalidSetExCommand = "ERROR: Invalid SETEX command. Format: SETEX <key> <value> <ttl_seconds>"
	InvalidGetCommand   = "ERROR: Invalid GET command. Format: GET <key>"
	InvalidStatsCommand = "ERROR: Invalid STATS command. Format: STATS"
	UknownCommand       = "ERROR: Invalid command. Known commands: SET, GET, SETEX"
	InvalidTTLValue     = "ERROR: TTL must be a non-negative integer"
)

type Metrics struct {
	mu            sync.RWMutex
	ActiveClients int
	SetCount      int
	GetCount      int
	SetExCount    int
	ErrorCount    int
}

var kv = kvstore.New()
var connections = make(map[net.Conn]struct{})
var metrics = Metrics{}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	metrics.ActiveClients++

	conn.SetReadDeadline(time.Now().Add(Timeout * time.Second))
	conn.SetWriteDeadline(time.Now().Add(Timeout * time.Second))

	connections[conn] = struct{}{}
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		conn.SetReadDeadline(time.Now().Add(Timeout * time.Second))
		if err != nil {
			if err == io.EOF {
				log.Println("[INFO] Client disconnected:", getAddress(conn))
				disconnect(conn)
				return
			}

			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() {
				log.Println("[INFO] Client connection timed out:", getAddress(conn))
				disconnect(conn)
				return
			}

			log.Printf("[ERROR] Unable to read from %s: %v\n", getAddress(conn), err)
			disconnect(conn)
			return
		}

		message = strings.TrimSpace(message)
		tokens := strings.Split(message, " ")

		response := processCommand(tokens)
		response += "\nEND\n"

		_, err = conn.Write([]byte(response))
		conn.SetWriteDeadline(time.Now().Add(Timeout * time.Second))
		if err != nil {
			log.Printf("[ERROR] Error writing to %s: %v\n", getAddress(conn), err)
			disconnect(conn)
			return
		}
	}
}

func processCommand(tokens []string) string {
	if len(tokens) == 0 {
		log.Println("[WARN] Received empty command")
		metrics.ErrorCount++
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
			metrics.ErrorCount++
			return InvalidGetCommand
		}
		log.Printf("[INFO] GET %s -> %s\n", key, value)
		metrics.GetCount++
		return value
	case SetCommand:
		if len(tokens) != 3 {
			log.Println("[WARN] Invalid SET command format")
			metrics.ErrorCount++
			return InvalidSetCommand
		}
		key, value := tokens[1], tokens[2]
		kv.Set(key, value)
		log.Printf("[INFO] SET %s %s -> OK\n", key, value)
		metrics.SetCount++
		return PutOK
	case SetexCommand:
		if len(tokens) != 4 {
			log.Println("[WARN] Invalid SETEX command format")
			metrics.ErrorCount++
			return InvalidSetExCommand
		}
		key, value, ttlStr := tokens[1], tokens[2], tokens[3]

		ttl, err := strconv.Atoi(ttlStr)
		if err != nil || ttl <= 0 {
			log.Println("[WARN] TTL in SETEX is not a positive integer")
			metrics.ErrorCount++
			return InvalidTTLValue
		}

		kv.SetEx(key, value, ttl)
		log.Printf("[INFO] SETEX %s %s (TTL: %d) -> OK\n", key, value, ttl)
		metrics.SetExCount++
		return PutOK
	case StatsCommand:
		if len(tokens) != 1 {
			log.Println("[WARN] Invalid STATS command format")
			metrics.ErrorCount++
			return InvalidStatsCommand
		}
		return statsString()
	default:
		log.Printf("[WARN] Unknown command: %s\n", command)
		metrics.ErrorCount++
		return UknownCommand
	}
}

func getAddress(conn net.Conn) string {
	return conn.RemoteAddr().String()
}

func setupShutdownHook(ln net.Listener) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("[INFO] Shutting down server...")
		ln.Close()

		for conn := range connections {
			conn.Close()
		}
	}()
}

func disconnect(conn net.Conn) {
	conn.Close()
	delete(connections, conn)
	metrics.ActiveClients--
}

func statsString() string {
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()

	return fmt.Sprintf(
		"Active clients: %d\nSET: %d\nGET: %d\nSETEX: %d\nErrors: %d",
		metrics.ActiveClients,
		metrics.SetCount,
		metrics.GetCount,
		metrics.SetExCount,
		metrics.ErrorCount,
	)
}

func StartServer() {
	ln, err := net.Listen("tcp", Port)
	if err != nil {
		log.Fatalf("[FATAL] Failed to start server: %v\n", err)
		return
	}
	setupShutdownHook(ln)
	defer ln.Close()
	log.Println("[INFO] Server is listening on port 8080...")

	// Main loop
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[INFO] Listener closed: %v\n", err)
			break
		}
		log.Println("[INFO] Client connected:", getAddress(conn))
		go handleConnection(conn)
	}
}
