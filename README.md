### KVStore: Lightweight In-Memory Key-Value Store in Go

A simple TCP-based key-value store written in Go, supporting basic commands like SET, GET, SETEX (with TTL), and STATS.

⸻

### Features
	•	SET <key> <value> – Store a key-value pair
	•	GET <key> – Retrieve the value for a key
	•	SETEX <key> <value> <ttl_seconds> – Set key with automatic expiration
	•	STATS – Returns basic metrics (command counts, active clients, errors)
	•	Graceful shutdown on Ctrl+C (SIGINT)
	•	Handles multiple clients concurrently
	•	Includes a stress test script to simulate load

⸻

### Project Structure

kvstore/
├── client/         # CLI client that connects to the server
│   └── main.go
├── server/         # TCP server implementation
│   └── main.go
├── kvstore/        # Core key-value store logic (with TTL support)
│   └── kvstore.go
├── stress/         # Stress testing script
│   └── stress.go
├── go.mod
└── README.md



⸻

### How to Run

✅ Start the Server

go run ./server

✅ Start the Client

go run ./client

✅ Try Commands

kv> SET foo bar
OK

kv> GET foo
bar

kv> SETEX temp value 5
OK

kv> STATS
Active clients: 1
SET: 1
GET: 1
SETEX: 1
Errors: 0

✅ Run Stress Test

go run ./stress



⸻

🧪 Command Summary

Command	Description
SET key value	Stores the value
GET key	Retrieves the value
SETEX k v ttl	Stores value with expiration in seconds
STATS	Shows internal server metrics



⸻

🛡 Graceful Shutdown
	•	Pressing Ctrl+C triggers a clean shutdown:
	•	Stops accepting new connections
	•	Closes existing listener
	•	Logs shutdown status

⸻

📌 Future Ideas
	•	Add persistence (write to file or database)
	•	Support DEL, EXISTS, or KEYS commands
	•	Web-based metrics endpoint (Prometheus-friendly)
	•	Authentication support

⸻

📄 License

MIT

⸻

Let me know if you’d like this exported to a file or want to include code snippets for each command.