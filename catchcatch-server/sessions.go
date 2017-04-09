package main

import (
	"errors"
	"log"
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
	conns := sm.copyConns()
	conns[id] = conn
	sm.connections.Store(conns)
}

// Remove engineio.Conn by session id
func (sm *SessionManager) Remove(id string) {
	conns := sm.copyConns()
	delete(conns, id)
	sm.connections.Store(conns)
}

// Emit send payload on eventX to socket id
func (sm *SessionManager) Emit(id, event string, payload string) error {
	conn := sm.Get(id)
	if conn == nil {
		return errors.New("connection not found")
	}
	log.Println("Send event to", id, event, payload)
	writer, err := conn.NextWriter(engineio.MessageText)
	if err != nil {
		return errors.New("error sent message to " + id + ":" + err.Error())
	}
	msg := []byte(`2["` + event + `",` + payload + "]")
	if _, err := writer.Write(msg); err != nil {
		return err
	}
	return writer.Close()
}

// CloseAll engineio.Conn
func (sm *SessionManager) CloseAll() {
	conns := sm.connections.Load().(connStore)
	for _, c := range conns {
		c.Close()
	}
}

func (sm *SessionManager) copyConns() connStore {
	conns := sm.connections.Load().(connStore)
	newConns := make(connStore)
	for k, v := range conns {
		newConns[k] = v
	}
	return newConns
}
