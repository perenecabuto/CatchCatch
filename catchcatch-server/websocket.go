package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/protobuf"
	uuid "github.com/satori/go.uuid"
)

// WebSocketConnection is an interface for websocket communication
type WebSocketConnection interface {
	Read(*[]byte) (int, error)
	Send(payload []byte) error
	Close() error
}

// WebSocketDriver is an interface for websocket connection handling
type WebSocketDriver interface {
	Handler(ctx context.Context, onConnect func(context.Context, WebSocketConnection)) http.Handler
}

// Conn represents a websocket connection
type Conn struct {
	WebSocketConnection

	ID             string
	eventCallbacks map[string]evtCallback
	onDisconnected func()
	stopFunc       context.CancelFunc

	buffer []byte
}

type evtCallback func([]byte)

func (c *Conn) listen(ctx context.Context) error {
	ctx, c.stopFunc = context.WithCancel(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := c.readMessage(); err != nil {
				return err
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

// Message represent protobuf message with event name
type Message interface {
	proto.Message
	GetEventName() string
}

// Emit send payload on eventX to socket id
func (c *Conn) Emit(message Message) error {
	payload, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	return c.Send(payload)
}

func (c *Conn) close() {
	c.Close()
	c.stopFunc()
	go c.onDisconnected()
}

func (c *Conn) readMessage() error {
	length, err := c.Read(&c.buffer)
	if err != nil {
		return err
	}
	if length == 0 {
		return nil
	}
	msg := &protobuf.Simple{}
	if err := proto.Unmarshal(c.buffer[:length], msg); err != nil {
		return fmt.Errorf("readMessage(unmarshall): %s %s", err.Error(), c.buffer[:length])
	}
	if len(msg.String()) == 0 {
		log.Println("message error:", msg)
		return fmt.Errorf("Invalid payload: %s", c.buffer)
	}
	cb, exists := c.eventCallbacks[msg.GetEventName()]
	if !exists {
		return fmt.Errorf("No callback found for: %v", msg)
	}
	return withRecover(func() error {
		cb(c.buffer)
		return nil
	})
}

// WebSocketServer manage websocket connections
type WebSocketServer struct {
	handler     WebSocketDriver
	onConnected func(c *Conn)

	connections atomic.Value
	sync.Mutex
}

type connectionGroup map[string]*Conn

// NewWebSocketServer create a new WebSocketServer
func NewWebSocketServer(handler WebSocketDriver) *WebSocketServer {
	wss := &WebSocketServer{handler: handler, onConnected: func(c *Conn) {}}
	wss.connections.Store(make(connectionGroup))
	return wss
}

// OnConnected register event callback to new connections
func (wss *WebSocketServer) OnConnected(fn func(c *Conn)) {
	if fn != nil {
		wss.onConnected = fn
	}
}

// Listen to websocket connections
func (wss *WebSocketServer) Listen(ctx context.Context) http.Handler {
	return wss.handler.Handler(ctx, func(ctx context.Context, c WebSocketConnection) {
		conn := wss.Add(c)
		err := withRecover(func() error {
			wss.onConnected(conn)
			return conn.listen(ctx)
		})
		if err != nil {
			log.Println("WebSocketServer: read error", err)
		}
		wss.Remove(conn.ID)
	})
}

// Get Conn by session id
func (wss *WebSocketServer) Get(id string) *Conn {
	connections := wss.connections.Load().(connectionGroup)
	return connections[id]
}

// Add Conn for session id
func (wss *WebSocketServer) Add(c WebSocketConnection) *Conn {
	id := uuid.NewV4().String()
	conn := &Conn{c, id, make(map[string]evtCallback), func() {}, func() {}, make([]byte, 512)}
	wss.withConnections(func(connections connectionGroup) {
		connections[conn.ID] = conn
	})
	return conn
}

func (wss *WebSocketServer) withConnections(fn func(connectionGroup)) {
	wss.Lock()
	connections := wss.connections.Load().(connectionGroup)
	fn(connections)
	newGroup := make(connectionGroup)
	for k, v := range connections {
		newGroup[k] = v
	}
	wss.connections.Store(newGroup)
	wss.Unlock()
}

// Remove Conn by session id
func (wss *WebSocketServer) Remove(id string) {
	if c := wss.Get(id); c != nil {
		c.close()
		wss.withConnections(func(connections connectionGroup) {
			delete(connections, id)
		})
	}
}

// Emit send payload on eventX to socket id
func (wss *WebSocketServer) Emit(id string, message Message) error {
	if conn := wss.Get(id); conn != nil {
		conn.Emit(message)
		return nil
	}
	return errors.New("connection not found")
}

// BroadcastTo ids event message
func (wss *WebSocketServer) BroadcastTo(ids []string, message Message) {
	for _, id := range ids {
		if err := wss.Emit(id, message); err != nil {
			log.Println("error to emit ", message, message, err)
		}
	}
}

// Broadcast event message to all connections
func (wss *WebSocketServer) Broadcast(message Message) error {
	connections := wss.connections.Load().(connectionGroup)
	for id := range connections {
		if err := wss.Emit(id, message); err != nil {
			return err
		}
	}
	return nil
}

// CloseAll Conn
func (wss *WebSocketServer) CloseAll() {
	connections := wss.connections.Load().(connectionGroup)
	for _, c := range connections {
		c.close()
	}
}
