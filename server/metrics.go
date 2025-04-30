package server

import (
	"strings"
	"sync"
)

type Metrics struct {
	mu            sync.RWMutex
	ActiveClients int
	CommandCounts map[string]int
}

// NewMetrics creates and initializes the Metrics struct
func NewMetrics() *Metrics {
	return &Metrics{
		CommandCounts: make(map[string]int),
	}
}

func (m *Metrics) Inc(command string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := strings.ToUpper(command)
	if _, exists := m.CommandCounts[key]; !exists {
		m.CommandCounts[key] = 1
	} else {
		m.CommandCounts[key]++
	}
}

func (m *Metrics) Get(command string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count, exists := m.CommandCounts[strings.ToUpper(command)]
	if !exists {
		return 0
	}
	return count
}

// TotalCommands returns the sum of all successful commands (excluding "ERROR")
func (m *Metrics) TotalCommands() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0
	for cmd, count := range m.CommandCounts {
		if cmd != "ERROR" {
			total += count
		}
	}
	return total
}

// IncActiveClients safely increments ActiveClients
func (m *Metrics) IncActiveClients() {
	m.mu.Lock()
	m.ActiveClients++
	m.mu.Unlock()
}

// DecActiveClients safely decrements ActiveClients
func (m *Metrics) DecActiveClients() {
	m.mu.Lock()
	m.ActiveClients--
	m.mu.Unlock()
}

// Snapshot returns a copy of the current metrics
func (m *Metrics) Snapshot() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Deep copy of the map
	countsCopy := make(map[string]int, len(m.CommandCounts))
	for k, v := range m.CommandCounts {
		countsCopy[k] = v
	}

	return Metrics{
		ActiveClients: m.ActiveClients,
		CommandCounts: countsCopy,
	}
}
