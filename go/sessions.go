package main

import "sync/atomic"

type Connections map[string]engineio.Conn

type SessionManager struct {
	connections atomic.Value
}

func NewSessionManager() *SessionManager {
	sessions := &SessionManager{}
	sessions.connections.Store(make(Connections))
	return sessions
}

func (sm *SessionManager) Get(id string) engineio.Conn {
	conns := sm.connections.Load().(Connections)
	return conns[id]
}

func (sm *SessionManager) Set(id string, conn engineio.Conn) {
	conns := sm.connections.Load().(Connections)
	conns[id] = conn
	sm.connections.Store(conns)
}

func (sm *SessionManager) Remove(id string) {
	conns := sm.connections.Load().(Connections)
	delete(conns, id)
	sm.connections.Store(conns)
}
