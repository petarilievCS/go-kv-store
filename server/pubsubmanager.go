package server

import (
	"fmt"
	"net"
	"sync"
)

type PubSubManager struct {
	mu            sync.RWMutex
	Subscribtions map[string]map[net.Conn]bool
}

func NewPubSubManager() *PubSubManager {
	return &PubSubManager{
		Subscribtions: make(map[string]map[net.Conn]bool),
	}
}

func (m *PubSubManager) Subscribe(channel string, conn net.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Subscribtions[channel] == nil {
		m.Subscribtions[channel] = make(map[net.Conn]bool)
	}
	connections := m.Subscribtions[channel][conn] = true
}

func (m *PubSubManager) Unsubscribe(channel string, conn net.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	connections, exists := m.Subscribtions[channel]
	if exists {
		delete(connections, conn)
		if len(connections) == 0 {
			delete(m.Subscribtions, channel)
		}
	}
}

func (m *PubSubManager) Publish(channel string, message string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connections, exists := m.Subscribtions[channel]
	if !exists {
		return
	}

	count := 0
	for conn := range connections {
		_, err := fmt.Fprintf(conn, message+"\n")
		if err != 0 {
			count++
		}
	}
	return count
}
