package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"

	engineio "github.com/googollee/go-engine.io"
)

type conn struct {
	id   string
	conn engineio.Conn
}
type connStore map[string]conn

// SessionManager manage websocket connections
type SessionManager struct {
	connections connStore
	addCH       chan conn
	delCH       chan string
}

// NewSessionManager create a new SessionManager
func NewSessionManager(ctx context.Context) *SessionManager {
	sessions := &SessionManager{make(connStore), make(chan conn), make(chan string)}
	go sessions.watchConnections(ctx)
	return sessions
}

func (sm *SessionManager) watchConnections(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(sm.addCH)
			close(sm.delCH)
			return
		case conn := <-sm.addCH:
			sm.connections[conn.id] = conn
		case id := <-sm.delCH:
			delete(sm.connections, id)
		}
	}
}

// Get engineio.Conn by session id
func (sm *SessionManager) Get(id string) engineio.Conn {
	if conn, exists := sm.connections[id]; exists {
		return conn.conn
	}
	return nil
}

// Set engineio.Conn for session id
func (sm *SessionManager) Set(id string, c engineio.Conn) {
	sm.addCH <- conn{id, c}
}

// Remove engineio.Conn by session id
func (sm *SessionManager) Remove(id string) {
	sm.delCH <- id
}

// Emit send payload on eventX to socket id
func (sm *SessionManager) Emit(id, event string, message interface{}) error {
	payload, err := messagePayload(message)
	if err != nil {
		return err
	}
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

// BroadcastTo ids event message
func (sm *SessionManager) BroadcastTo(ids []string, event string, message interface{}) {
	for _, id := range ids {
		if err := sm.Emit(id, event, message); err != nil {
			log.Println("error to emit "+event, message, err)
		}
	}
}

// CloseAll engineio.Conn
func (sm *SessionManager) CloseAll() {
	for _, c := range sm.connections {
		c.conn.Close()
	}
}

func messagePayload(msg interface{}) (string, error) {
	switch msg.(type) {
	case string:
		return strconv.Quote(msg.(string)), nil
	default:
		jPayload, err := json.Marshal(msg)
		if err != nil {
			return "", err
		}
		return string(jPayload), nil
	}
}
