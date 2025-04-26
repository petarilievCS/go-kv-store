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
	"syscall"
	"time"

	"github.com/petariliev/kvstore/kvstore"
)

const (
	OK            = "OK"
	GetCommand    = "GET"
	SetCommand    = "SET"
	SetexCommand  = "SETEX"
	StatsCommand  = "STATS"
	DeleteCommand = "DELETE"
	KeysCommand   = "KEYS"
	Port          = ":8080"
	Timeout       = 30
	FileName      = "data.txt"
)

// Errors
const (
	InvalidCommand       = "ERROR: Invalid command. Known commands: SET, GET, SETEX"
	InvalidSetCommand    = "ERROR: Invalid SET command. Format: SET <key> <value>"
	InvalidSetExCommand  = "ERROR: Invalid SETEX command. Format: SETEX <key> <value> <ttl_seconds>"
	InvalidGetCommand    = "ERROR: Invalid GET command. Format: GET <key>"
	InvalidStatsCommand  = "ERROR: Invalid STATS command. Format: STATS"
	InvalidDeleteCommand = "ERROR: Invalid DELETE command. Format: DELETE <key>"
	UknownCommand        = "ERROR: Invalid command. Known commands: SET, GET, SETEX"
	InvalidKeysCommand   = "ERROR: Invalid KEYS command. Format: KEYS"
	InvalidTTLValue      = "ERROR: TTL must be a non-negative integer"
)

var kv = kvstore.New()
var connections = NewConnections()
var metrics = Metrics{}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	metrics.IncActiveClients()

	conn.SetReadDeadline(time.Now().Add(Timeout * time.Second))
	conn.SetWriteDeadline(time.Now().Add(Timeout * time.Second))

	connections.Add(conn)
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
		metrics.IncError()
		return InvalidCommand
	}

	switch tokens[0] {
	case GetCommand:
		return handleGet(tokens)
	case SetCommand:
		return handleSet(tokens)
	case SetexCommand:
		return handleSetEx(tokens)
	case StatsCommand:
		return handleStats(tokens)
	case DeleteCommand:
		return handleDelete(tokens)
	case KeysCommand:
		return handleKeys(tokens)
	default:
		log.Printf("[WARN] Unknown command: %s\n", tokens[0])
		metrics.IncError()
		return UknownCommand
	}
}

// Command handlers
func handleGet(tokens []string) string {
	if len(tokens) != 2 {
		log.Println("[WARN] Invalid GET command format")
		metrics.IncError()
		return InvalidGetCommand
	}
	key := tokens[1]
	value, err := kv.Get(key)
	if err != nil {
		log.Printf("[WARN] GET %s -> key not found\n", key)
		metrics.IncError()
		return kvstore.KeyNotFound
	}
	log.Printf("[INFO] GET %s -> %s\n", key, value)
	metrics.IncGet()
	return value
}

func handleSet(tokens []string) string {
	if len(tokens) != 3 {
		log.Println("[WARN] Invalid SET command format")
		metrics.IncError()
		return InvalidSetCommand
	}
	key, value := tokens[1], tokens[2]
	kv.Set(key, value)
	log.Printf("[INFO] SET %s %s -> OK\n", key, value)
	metrics.IncSet()
	return OK
}

func handleSetEx(tokens []string) string {
	if len(tokens) != 4 {
		log.Println("[WARN] Invalid SETEX command format")
		metrics.IncError()
		return InvalidSetExCommand
	}
	key, value, ttlStr := tokens[1], tokens[2], tokens[3]

	ttl, err := strconv.Atoi(ttlStr)
	if err != nil || ttl <= 0 {
		log.Println("[WARN] TTL in SETEX is not a positive integer")
		metrics.IncError()
		return InvalidTTLValue
	}

	kv.SetEx(key, value, ttl)
	log.Printf("[INFO] SETEX %s %s (TTL: %d) -> OK\n", key, value, ttl)
	metrics.IncSetEx()
	return OK
}

func handleStats(tokens []string) string {
	if len(tokens) != 1 {
		log.Println("[WARN] Invalid STATS command format")
		metrics.IncError()
		return InvalidStatsCommand
	}
	return statsString()
}

func handleDelete(tokens []string) string {
	if len(tokens) != 2 {
		log.Println("[WARN] Invalid DELETE command format")
		metrics.IncError()
		return InvalidDeleteCommand
	}
	key := tokens[1]
	err := kv.Delete(key)
	if err != nil {
		log.Printf("[WARN] GET %s -> key not found\n", key)
		metrics.IncError()
		return kvstore.KeyNotFound
	}
	metrics.IncDelete()
	log.Printf("[INFO] DELETE %s -> OK", tokens[1])
	return OK
}

func handleKeys(tokens []string) string {
	if len(tokens) != 1 {
		log.Println("[WARN] Invalid KEYS command format")
		metrics.IncError()
		return InvalidKeysCommand
	}

	keys := kv.Keys()
	metrics.IncKeys()
	log.Printf("[INFO] KEYS -> %v\n", keys)

	if len(keys) == 0 {
		return "EMPTY"
	}
	return strings.Join(keys, "\n")
}

// Helper methods\
func getAddress(conn net.Conn) string {
	return conn.RemoteAddr().String()
}

func setupShutdownHook(ln net.Listener) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("[INFO] Shutting down server...")
		connections.CloseAll()

		log.Println("[INFO] Saving data to disk...")
		err := kv.SaveToDisk(FileName)
		if err != nil {
			log.Printf("[ERROR] Error while saving data to disk: %s\n", err)
		}

		ln.Close()
	}()
}

func disconnect(conn net.Conn) {
	conn.Close()
	connections.Remove(conn)
	metrics.DecActiveClients()
}

func statsString() string {
	snapshot := metrics.Snapshot()

	return fmt.Sprintf(
		"Active clients: %d\nSET: %d\nGET: %d\nSETEX: %d\nDELETE: %d\nKEYS: %d\nErrors: %d",
		snapshot.ActiveClients,
		snapshot.SetCount,
		snapshot.GetCount,
		snapshot.SetExCount,
		snapshot.DeleteCount,
		snapshot.KeysCount,
		snapshot.ErrorCount,
	)
}

// Main method
func StartServer() {
	log.Println("[INFO] Starting server...")
	log.Println("[INFO] Loading data from disk...")

	err := kv.LoadFromDisk(FileName)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[INFO] File %s does not exist, likely first startup\n", FileName)
		} else {
			log.Printf("[ERROR] Error loading data from disk: %s\n", err)
		}
	} else {
		log.Println("[INFO] Loaded data from disk")
	}

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
