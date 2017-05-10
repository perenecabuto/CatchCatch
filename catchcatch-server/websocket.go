package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	websocket "github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

// Conn represents a websocket connection
type Conn struct {
	ID   string
	conn *websocket.Conn

	eventCallbacks map[string]evtCallback
	onDisconnected func()
	stopFunc       context.CancelFunc
}

const (
	sizeOfMessageBuffer = 1024
)

// NewConn creates ws client connection handler
func NewConn(conn *websocket.Conn) *Conn {
	id := uuid.NewV4().String()
	return &Conn{ID: id, conn: conn, eventCallbacks: make(map[string]evtCallback)}
}

type evtCallback func(string)

func (c *Conn) listen(ctx context.Context, doneFunc func(error)) {
	ctx, c.stopFunc = context.WithCancel(ctx)
	for {
		select {
		case <-ctx.Done():
			doneFunc(nil)
			return
		default:
			if err := c.readMessage(); err != nil {
				doneFunc(err)
				return
			}
		}
	}
}

// On this connection event trigger callback with its message
func (c *Conn) On(event string, callback evtCallback) {
	c.eventCallbacks[event] = callback
}

// OnDisconnected register event callback to closed connections
func (c *Conn) OnDisconnected(fn func()) {
	if fn != nil {
		c.onDisconnected = fn
	}
}

// Emit send payload on eventX to socket id
func (c *Conn) Emit(event string, message interface{}) error {
	payload, err := parsePayload(message)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, []byte(event+","+payload))
}

func (c *Conn) close() {
	c.stopFunc()
	c.conn.Close()
	go c.onDisconnected()
}

func (c *Conn) readMessage() error {
	_, r, err := c.conn.NextReader()
	if err != nil {
		return err
	}

	messagebuf, n := make([]byte, sizeOfMessageBuffer), 0
	if n, err = r.Read(messagebuf); err != nil {
		return err
	}
	data := strings.SplitN(string(messagebuf[:n]), ",", 2)
	if len(data) == 0 {
		log.Println("message error:", messagebuf)
		return errors.New("Invalid payload: " + string(messagebuf))
	}
	if cb, exists := c.eventCallbacks[data[0]]; exists {
		log.Println("gotten data:", data[1])
		cb(data[1])
		return nil
	}
	return fmt.Errorf("No callback found: %v", data)
}

// WebSocketServer manage websocket connections
type WebSocketServer struct {
	connections map[string]*Conn
	onConnected func(c *Conn)
	upgrader    websocket.Upgrader
	sync.RWMutex
}

// NewWebSocketServer create a new WebSocketServer
func NewWebSocketServer(ctx context.Context) *WebSocketServer {
	return &WebSocketServer{
		connections: make(map[string]*Conn),
		onConnected: func(c *Conn) {},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  sizeOfMessageBuffer,
			WriteBufferSize: sizeOfMessageBuffer,
		},
	}
}

// OnConnected register event callback to new connections
func (wss *WebSocketServer) OnConnected(fn func(c *Conn)) {
	if fn != nil {
		wss.onConnected = fn
	}
}

// Listen to websocket connections
func (wss *WebSocketServer) Listen(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := wss.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		conn := wss.Add(c)
		wss.onConnected(conn)
		conn.listen(ctx, func(err error) {
			if err != nil {
				log.Println("WebSocketServer: exit - read error", err)
			}
		})
		wss.Remove(conn.ID)
	}
}

// Add Conn for session id
func (wss *WebSocketServer) Add(c *websocket.Conn) *Conn {
	conn := NewConn(c)
	wss.Lock()
	wss.connections[conn.ID] = conn
	wss.Unlock()
	return conn
}

// Remove Conn by session id
func (wss *WebSocketServer) Remove(id string) {
	wss.Lock()
	if c, exists := wss.connections[id]; exists {
		c.close()
		delete(wss.connections, id)
	}
	wss.Unlock()
}

// Emit send payload on eventX to socket id
func (wss *WebSocketServer) Emit(id, event string, message interface{}) error {
	wss.RLock()
	defer wss.RUnlock()
	if conn := wss.connections[id]; conn != nil {
		conn.Emit(event, message)
		return nil
	}
	return errors.New("connection not found")
}

// BroadcastTo ids event message
func (wss *WebSocketServer) BroadcastTo(ids []string, event string, message interface{}) {
	for _, id := range ids {
		if err := wss.Emit(id, event, message); err != nil {
			log.Println("error to emit "+event, message, err)
		}
	}
}

// Broadcast event message to all connections
func (wss *WebSocketServer) Broadcast(event string, message interface{}) {
	for id := range wss.connections {
		if err := wss.Emit(id, event, message); err != nil {
			log.Println("error to emit "+event, message, err)
		}
	}
}

// CloseAll Conn
func (wss *WebSocketServer) CloseAll() {
	for _, c := range wss.connections {
		c.conn.Close()
	}
}

func parsePayload(msg interface{}) (string, error) {
	switch msg.(type) {
	case string:
		return msg.(string), nil
	default:
		jPayload, err := json.Marshal(msg)
		if err != nil {
			return "", err
		}
		return string(jPayload), nil
	}
}
