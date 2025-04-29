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
	DeleteCount   int
	DeleteExCount int
	FlushCount    int
	KeysCount     int
	PingCount     int
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

func (m *Metrics) IncDelete() {
	m.mu.Lock()
	m.DeleteCount++
	m.mu.Unlock()
}

func (m *Metrics) IncDeleteEx() {
	m.mu.Lock()
	m.DeleteExCount++
	m.mu.Unlock()
}

func (m *Metrics) IncFlush() {
	m.mu.Lock()
	m.FlushCount++
	m.mu.Unlock()
}

func (m *Metrics) IncKeys() {
	m.mu.Lock()
	m.KeysCount++
	m.mu.Unlock()
}

func (m *Metrics) IncPing() {
	m.mu.Lock()
	m.PingCount++
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
		DeleteCount:   m.DeleteCount,
		DeleteExCount: m.DeleteExCount,
		FlushCount:    m.FlushCount,
		KeysCount:     m.KeysCount,
		PingCount:     m.PingCount,
		ErrorCount:    m.ErrorCount,
	}
}
