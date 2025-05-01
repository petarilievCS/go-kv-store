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
	OK               = "OK"
	GetCommand       = "GET"
	MGetCommand      = "MGET"
	KeyExistsCommand = "KEYEXISTS"
	TypeCommand      = "TYPE"
	SetCommand       = "SET"
	MSetCommand      = "MSET"
	SetexCommand     = "SETEX"
	TTLCommand       = "TTL"
	RenameCommand    = "RENAME"
	StatsCommand     = "STATS"
	DeleteCommand    = "DELETE"
	DelCommand       = "DEL"
	DeleteexCommand  = "DELETEEX"
	FlushCommand     = "FLUSH"
	SaveCommand      = "SAVE"
	LoadCommand      = "LOAD"
	KeysCommand      = "KEYS"
	InfoCommand      = "INFO"
	HelpCommand      = "HELP"
	PingCommand      = "PING"
	ShutDownCommand  = "SHUTDOWN"
	Port             = ":8080"
	Timeout          = 30
	FileName         = "data.txt"
	InvalidCommand   = "ERROR: Invalid command."
	ServerVersion    = "1.0.0"
)

var kv = kvstore.New()
var connections = NewConnections()
var metrics = NewMetrics()
var done = make(chan struct{})
var startTime = time.Now()

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
		metrics.Inc("ERROR")
		return InvalidCommand
	}

	cmd := strings.ToUpper(tokens[0])
	switch cmd {
	case GetCommand:
		return handleGet(tokens)
	case MGetCommand:
		return handleMGet(tokens)
	case KeyExistsCommand:
		return handleKeyExists(tokens)
	case TypeCommand:
		return handleType(tokens)
	case SetCommand:
		return handleSet(tokens)
	case MSetCommand:
		return handleMSet(tokens)
	case SetexCommand:
		return handleSetEx(tokens)
	case TTLCommand:
		return handleTTL(tokens)
	case RenameCommand:
		return handleRename(tokens)
	case StatsCommand:
		return handleStats(tokens)
	case DeleteCommand:
		return handleDelete(tokens)
	case DelCommand:
		return handleDel(tokens)
	case DeleteexCommand:
		return handleDeleteEx(tokens)
	case FlushCommand:
		return handleFlush(tokens)
	case SaveCommand:
		return handleSave(tokens)
	case LoadCommand:
		return handleLoad(tokens)
	case KeysCommand:
		return handleKeys(tokens)
	case InfoCommand:
		return handleInfo(tokens)
	case HelpCommand:
		return handleHelp(tokens)
	case PingCommand:
		return handlePing(tokens)
	case ShutDownCommand:
		return handleShutDown(tokens)
	default:
		log.Printf("[WARN] Invalid command: %s\n", cmd)
		metrics.Inc("ERROR")
		return InvalidCommand
	}
}

// Command handlers
func handleGet(tokens []string) string {
	if len(tokens) != 2 {
		log.Println("[WARN] Invalid GET command format")
		metrics.Inc("ERROR")
		return formatInvalidCommand("GET", "GET <key>")
	}
	key := tokens[1]
	value, err := kv.Get(key)
	if err != nil {
		log.Printf("[WARN] GET %s -> key not found\n", key)
		metrics.Inc("ERROR")
		return kvstore.KeyNotFound
	}
	log.Printf("[INFO] GET %s -> %s\n", key, value)
	metrics.Inc("GET")
	return value
}

func handleMGet(tokens []string) string {
	if len(tokens) < 2 {
		metrics.Inc("ERROR")
		return formatInvalidCommand("MGET", "MGET <key1> <key2> ...")
	}

	var sb strings.Builder
	for _, key := range tokens[1:] {
		value, err := kv.Get(key)
		if err != nil {
			sb.WriteString("nil\n")
		} else {
			sb.WriteString(value + "\n")
		}
	}

	log.Printf("[INFO] MGET %v\n", tokens[1:])
	metrics.Inc("MGET")
	return strings.TrimRight(sb.String(), "\n")
}

func handleKeyExists(tokens []string) string {
	if len(tokens) != 2 {
		metrics.Inc("ERROR")
		return formatInvalidCommand("KEYEXISTS", "KEYEXISTS <key>")
	}

	key := tokens[1]
	keyExists := kv.Contains(key)
	metrics.Inc("KEYEXISTS")

	if keyExists {
		log.Printf("[INFO] KEYEXISTS %s -> 1\n", key)
		return "1"
	}
	log.Printf("[INFO] KEYEXISTS %s -> 0\n", key)
	return "0"
}

func handleType(tokens []string) string {
	if len(tokens) != 2 {
		metrics.Inc("ERROR")
		return formatInvalidCommand("TYPE", "TYPE <key>")
	}

	key := tokens[1]
	if kv.Contains(key) {
		return "string"
	}
	metrics.Inc("TYPE")
	return "none"
}

func handleSet(tokens []string) string {
	if len(tokens) != 3 {
		log.Println("[WARN] Invalid SET command format")
		metrics.Inc("ERROR")
		return formatInvalidCommand("SET", "SET <key> <value>")
	}
	key, value := tokens[1], tokens[2]
	kv.Set(key, value)
	log.Printf("[INFO] SET %s %s -> OK\n", key, value)
	metrics.Inc("SET")
	return OK
}

func handleMSet(tokens []string) string {
	if len(tokens) < 3 || len(tokens)%2 != 1 {
		metrics.Inc("ERROR")
		return formatInvalidCommand("MSET", "MSET <key1> <val1> <key2> <val2> ...")
	}

	for i := 1; i < len(tokens); i += 2 {
		key, value := tokens[i], tokens[i+1]
		kv.Set(key, value)
	}

	log.Printf("[INFO] MSET -> %d keys set\n", len(tokens)/2)
	metrics.Inc("MSET")
	return OK
}

func handleSetEx(tokens []string) string {
	if len(tokens) != 4 {
		log.Println("[WARN] Invalid SETEX command format")
		metrics.Inc("ERROR")
		return formatInvalidCommand("SETEX", "SETEX <key> <value> <ttl_seconds>")
	}
	key, value, ttlStr := tokens[1], tokens[2], tokens[3]

	ttl, err := strconv.Atoi(ttlStr)
	if err != nil || ttl <= 0 {
		log.Println("[WARN] TTL in SETEX is not a positive integer")
		metrics.Inc("ERROR")
		return formatInvalidTTL(ttlStr)
	}

	kv.SetEx(key, value, ttl)
	log.Printf("[INFO] SETEX %s %s (TTL: %d) -> OK\n", key, value, ttl)
	metrics.Inc("SETEX")
	return OK
}

func handleTTL(tokens []string) string {
	if len(tokens) != 2 {
		metrics.Inc("ERROR")
		return formatInvalidCommand("TTL", "TTL <key>")
	}
	key := tokens[1]
	ttl := kv.TTL(key)

	switch ttl {
	case -2:
		log.Printf("[INFO] TTL %s -> key not found", key)
	case -1:
		log.Printf("[INFO] TTL %s -> no expiration", key)
	default:
		log.Printf("[INFO] TTL %s -> %d seconds", key, ttl)
	}

	metrics.Inc("TTL")
	return strconv.Itoa(ttl)
}

func handleRename(tokens []string) string {
	if len(tokens) != 3 {
		metrics.Inc("ERROR")
		return formatInvalidCommand("RENAME", "RENAME <oldKey> <newKey>")
	}

	oldKey, newKey := tokens[1], tokens[2]
	oldKeyExists := kv.Contains(oldKey)
	if !oldKeyExists {
		log.Printf("[WARN] RENAME %s -> key not found\n", oldKey)
		metrics.Inc("ERROR")
		return kvstore.KeyNotFound
	}

	kv.Rename(oldKey, newKey)
	log.Printf("[INFO] RENAME %s -> %s\n", oldKey, newKey)
	metrics.Inc("RENAME")
	return OK
}

func handleStats(tokens []string) string {
	if len(tokens) != 1 {
		log.Println("[WARN] Invalid STATS command format")
		metrics.Inc("ERROR")
		return formatInvalidCommand("STATS", "STATS")
	}
	return statsString()
}

func handleDelete(tokens []string) string {
	if len(tokens) != 2 {
		log.Println("[WARN] Invalid DELETE command format")
		metrics.Inc("ERROR")
		return formatInvalidCommand("DELETE", "DELETE <key>")
	}
	key := tokens[1]
	err := kv.Delete(key)
	if err != nil {
		log.Printf("[WARN] GET %s -> key not found\n", key)
		metrics.Inc("ERROR")
		return kvstore.KeyNotFound
	}
	metrics.Inc("DELETE")
	log.Printf("[INFO] DELETE %s -> OK\n", tokens[1])
	return OK
}

func handleDel(tokens []string) string {
	if len(tokens) < 2 {
		metrics.Inc("ERROR")
		return formatInvalidCommand("DEL", "DEL <key1> <key2> ...")
	}

	count := 0
	for _, key := range tokens[1:] {
		err := kv.Delete(key)
		if err == nil {
			count++
		}
	}
	log.Printf("[INFO] DEL %v -> %d keys deleted\n", tokens[1:], count)
	metrics.Inc("DEL")
	return strconv.Itoa(count)
}

func handleDeleteEx(tokens []string) string {
	if len(tokens) != 3 {
		log.Println("[WARN] Invalid DELETEX command format")
		metrics.Inc("ERROR")
		return formatInvalidCommand("DELETEEX", "DELETEEX <key> <ttl_seconds>")
	}

	key, delayStr := tokens[1], tokens[2]

	// Validate key
	_, err := kv.Get(key)
	if err != nil {
		log.Printf("[WARN] DELETEX %s %s -> key not found\n", key, delayStr)
		metrics.Inc("ERROR")
		return kvstore.KeyNotFound
	}

	// Validate time
	delay, err := strconv.Atoi(delayStr)
	if err != nil || delay <= 0 {
		log.Printf("[WARN] Time in DELETEX is not a positive integer: %s\n", delayStr)
		metrics.Inc("ERROR")
		return formatInvalidTTL(delayStr)
	}

	// Schedule deletion
	metrics.Inc("DELETEEX")
	time.AfterFunc(time.Duration(delay)*time.Second, func() {
		log.Printf("[INFO] DELETEEX %s %s -> OK\n", key, delayStr)
		kv.Delete(key)
	})
	return OK
}

func handleFlush(tokens []string) string {
	if len(tokens) != 1 {
		metrics.Inc("ERROR")
		return formatInvalidCommand("FLUSH", "FLUSH")
	}

	kv.Flush()
	log.Println("[INFO] FLUSH: store cleared")
	metrics.Inc("FLUSH")

	return OK
}

func handleSave(tokens []string) string {
	if len(tokens) != 1 {
		metrics.Inc("ERROR")
		return formatInvalidCommand("SAVE", "SAVE")
	}

	err := kv.SaveToDisk(FileName)
	if err != nil {
		log.Printf("[ERROR] Failed to save data: %v\n", err)
		metrics.Inc("ERROR")
		return fmt.Sprintf("ERROR: Failed to save to disk: %v", err)
	}

	log.Println("[INFO] SAVE: store saved to disk")
	metrics.Inc("SAVE")
	return OK
}

func handleLoad(tokens []string) string {
	if len(tokens) != 1 {
		metrics.Inc("ERROR")
		return formatInvalidCommand("LOAD", "LOAD")
	}

	err := kv.LoadFromDisk(FileName)
	if err != nil {
		log.Printf("[ERROR] Failed to load data: %v\n", err)
		metrics.Inc("ERROR")
		return fmt.Sprintf("ERROR: Failed to load data from disk: %v", err)
	}

	log.Println("[INFO] LOAD: loaded stroe from disk")
	metrics.Inc("LOAD")
	return OK
}

func handleKeys(tokens []string) string {
	if len(tokens) != 1 {
		log.Println("[WARN] Invalid KEYS command format")
		metrics.Inc("ERROR")
		return formatInvalidCommand("KEYS", "KEYS")
	}

	keys := kv.Keys()
	metrics.Inc("KEYS")
	log.Printf("[INFO] KEYS -> %v\n", keys)

	if len(keys) == 0 {
		return "EMPTY"
	}
	return strings.Join(keys, "\n")
}

func handleInfo(tokens []string) string {
	if len(tokens) != 1 {
		log.Println("[WARN] Invalid INFO command format")
		metrics.Inc("ERROR")
		return formatInvalidCommand("INFO", "INFO")
	}
	uptime := time.Since(startTime)

	metrics.mu.RLock()
	activeClients := metrics.ActiveClients
	metrics.mu.RUnlock()

	commandsProcessed := metrics.TotalCommands()
	keysInStore := len(kv.Keys())

	info := fmt.Sprintf(
		"Server Version: %s\n"+
			"Uptime: %s\n"+
			"Active Clients: %d\n"+
			"Total Commands Processed: %d\n"+
			"Keys in Store: %d\n",
		ServerVersion,
		uptime.Truncate(time.Second),
		activeClients,
		commandsProcessed,
		keysInStore,
	)

	metrics.Inc("INFO")
	log.Println("[INFO] INFO command requested")
	return info
}

func handleHelp(tokens []string) string {
	if len(tokens) != 1 {
		log.Println("[WARN] Invalid HELP command format")
		metrics.Inc("ERROR")
		return formatInvalidCommand("INFO", "INFO")
	}

	metrics.Inc("HELP")
	log.Println("[INFO] HELP command requested")
	return `Available commands:
	SET <key> <value>          - Store a key-value pair
	GET <key>                  - Retrieve a value
	SETEX <key> <value> <ttl>  - Store a key-value pair with expiration
	DELETE <key>               - Remove a key
	DELETEEX <key> <ttl>       - Remove a key after a delay
	KEYEXISTS <key>            - Check if a key exists
	FLUSH                      - Clear all keys
	KEYS                       - List all keys
	STATS                      - Show usage metrics
	INFO                       - Show server config
	PING                       - Check if server is alive
	SAVE                       - Save store to disk
	LOAD                       - Load store from disk
	SHUTDOWN                   - Gracefully stop the server
	HELP                       - Show this help message`
}

func handlePing(tokens []string) string {
	if len(tokens) != 1 {
		metrics.Inc("ERROR")
		return formatInvalidCommand("PING", "PING")
	}
	metrics.Inc("PING")
	return "PONG"
}

func handleShutDown(tokens []string) string {
	if len(tokens) != 1 {
		metrics.Inc("ERROR")
		return formatInvalidCommand("SHUTDOWN", "SHUTDOWN")
	}
	go triggerSIGINT()
	return "Server shutting down..."
}

// Helper methods
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

		close(done)
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

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Active clients: %d\n", snapshot.ActiveClients))

	tracked := []string{
		"SET", "GET", "SETEX", "DELETE", "DELETEEX", "KEYEXISTS", "FLUSH", "SAVE", "LOAD",
		"KEYS", "PING", "INFO", "HELP", "ERROR",
	}

	for _, cmd := range tracked {
		count := snapshot.Get(cmd)
		sb.WriteString(fmt.Sprintf("%s: %d\n", cmd, count))
	}

	return sb.String()
}

func formatInvalidCommand(cmd, expected string) string {
	return fmt.Sprintf("ERROR: Invalid %s command. Expected format: %s", cmd, expected)
}

func formatInvalidTTL(ttlStr string) string {
	return fmt.Sprintf("ERROR: Invalid TTL value '%s'. TTL must be a positive integer.", ttlStr)
}

func triggerSIGINT() {
	p, _ := os.FindProcess(os.Getpid())
	p.Signal(syscall.SIGINT)
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

	kv.ScheduleCleanup(10*time.Second, done)

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
