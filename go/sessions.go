package main

import (
	"sync/atomic"

	engineio "github.com/googollee/go-engine.io"
)

type connStore map[string]engineio.Conn

// SessionManager manage websocket connections
type SessionManager struct {
	connections atomic.Value
}

// NewSessionManager create a new SessionManager
func NewSessionManager() *SessionManager {
	sessions := &SessionManager{}
	sessions.connections.Store(make(connStore))
	return sessions
}

// Get engineio.Conn by session id
func (sm *SessionManager) Get(id string) engineio.Conn {
	conns := sm.connections.Load().(connStore)
	return conns[id]
}

// Set engineio.Conn for session id
func (sm *SessionManager) Set(id string, conn engineio.Conn) {
	conns := sm.connections.Load().(connStore)
	conns[id] = conn
	sm.connections.Store(conns)
}

// Remove engineio.Conn by session id
func (sm *SessionManager) Remove(id string) {
	conns := sm.connections.Load().(connStore)
	delete(conns, id)
	sm.connections.Store(conns)
}

// CloseAll engineio.Conn
func (sm *SessionManager) CloseAll() {
	conns := sm.connections.Load().(connStore)
	for _, c := range conns {
		c.Close()
	}
	sm.connections.Store(conns)
}
