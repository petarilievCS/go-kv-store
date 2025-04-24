package server

import (
	"net"
	"sync"
)

type Connections struct {
	mu    sync.RWMutex
	conns map[net.Conn]struct{}
}

func NewConnections() *Connections {
	return &Connections{
		conns: make(map[net.Conn]struct{}),
	}
}

func (p *Connections) Add(conn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.conns[conn] = struct{}{}
}

func (p *Connections) Remove(conn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.conns, conn)
}

func (p *Connections) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for conn := range p.conns {
		conn.Close()
	}
}
