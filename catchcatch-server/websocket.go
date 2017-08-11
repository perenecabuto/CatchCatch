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
	websocket "golang.org/x/net/websocket"
)

// Conn represents a websocket connection
type Conn struct {
	ID   string
	conn *websocket.Conn

	messagebuf     []byte
	eventCallbacks map[string]evtCallback
	onDisconnected func()
	stopFunc       context.CancelFunc
}

// NewConn creates ws client connection handler
func NewConn(conn *websocket.Conn) *Conn {
	id := uuid.NewV4().String()
	return &Conn{id, conn, make([]byte, 0), make(map[string]evtCallback), func() {}, func() {}}
}

type evtCallback func([]byte)

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

// Message represent protobuf message with event name
type Message interface {
	proto.Message
	GetId() string
	GetEventName() string
}

// Emit send payload on eventX to socket id
func (c *Conn) Emit(message Message) error {
	payload, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	return websocket.Message.Send(c.conn, payload)
}

func (c *Conn) close() {
	c.stopFunc()
	c.conn.Close()
	go c.onDisconnected()
}

func (c *Conn) readMessage() error {
	if err := websocket.Message.Receive(c.conn, &c.messagebuf); err != nil {
		log.Println("readMessage: " + err.Error())
		return err
	}

	msg := &protobuf.Simple{}
	proto.Unmarshal(c.messagebuf, msg)
	if len(msg.String()) == 0 {
		log.Println("message error:", c.messagebuf)
		return errors.New("Invalid payload: " + string(c.messagebuf))
	}
	if cb, exists := c.eventCallbacks[msg.GetEventName()]; exists {
		cb(c.messagebuf)
		return nil
	}
	return fmt.Errorf("No callback found for: %v", msg)
}

// WebSocketServer manage websocket connections
type WebSocketServer struct {
	connections atomic.Value
	onConnected func(c *Conn)

	sync.Mutex
}

type connectionGroup map[string]*Conn

// NewWebSocketServer create a new WebSocketServer
func NewWebSocketServer(ctx context.Context) *WebSocketServer {
	wss := &WebSocketServer{onConnected: func(c *Conn) {}}
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
	return websocket.Server{
		Handler: func(c *websocket.Conn) {
			conn := wss.Add(c)
			wss.onConnected(conn)
			conn.listen(ctx, func(err error) {
				if err != nil {
					log.Println("WebSocketServer: read error", err)
				}
			})
			wss.Remove(conn.ID)
		},
	}
}

// Get Conn by session id
func (wss *WebSocketServer) Get(id string) *Conn {
	connections := wss.connections.Load().(connectionGroup)
	return connections[id]
}

// Add Conn for session id
func (wss *WebSocketServer) Add(c *websocket.Conn) *Conn {
	conn := NewConn(c)
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
func (wss *WebSocketServer) Broadcast(message Message) {
	connections := wss.connections.Load().(connectionGroup)
	for id := range connections {
		if err := wss.Emit(id, message); err != nil {
			log.Println("error to emit ", message, err)
		}
	}
}

// CloseAll Conn
func (wss *WebSocketServer) CloseAll() {
	connections := wss.connections.Load().(connectionGroup)
	for _, c := range connections {
		c.conn.Close()
	}
}
