package server

import (
	"sync"
)

type Metrics struct {
	mu            sync.RWMutex
	ActiveClients int
	SetCount      int
	GetCount      int
	SetExCount    int
	ErrorCount    int
}

func (m *Metrics) IncActiveClients() {
	m.mu.Lock()
	m.ActiveClients++
	m.mu.Unlock()
}

func (m *Metrics) DecActiveClients() {
	m.mu.Lock()
	m.ActiveClients--
	m.mu.Unlock()
}

func (m *Metrics) IncSet() {
	m.mu.Lock()
	m.SetCount++
	m.mu.Unlock()
}

func (m *Metrics) IncGet() {
	m.mu.Lock()
	m.GetCount++
	m.mu.Unlock()
}

func (m *Metrics) IncSetEx() {
	m.mu.Lock()
	m.SetExCount++
	m.mu.Unlock()
}

func (m *Metrics) IncError() {
	m.mu.Lock()
	m.ErrorCount++
	m.mu.Unlock()
}

func (m *Metrics) Snapshot() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Metrics{
		ActiveClients: m.ActiveClients,
		SetCount:      m.SetCount,
		GetCount:      m.GetCount,
		SetExCount:    m.SetExCount,
		ErrorCount:    m.ErrorCount,
	}
}
