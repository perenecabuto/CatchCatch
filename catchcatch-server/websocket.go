package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/protobuf"
	uuid "github.com/satori/go.uuid"
)

var (
	// ErrWSConnectionNotFound is returned when connection id is not registred on this server
	ErrWSConnectionNotFound = errors.New("connection not found")
)

// WSConnection is an interface for WS communication
type WSConnection interface {
	Read(*[]byte) (int, error)
	Send(payload []byte) error
	Close() error
}

// WSDriver is an interface for WS communication
type WSDriver interface {
	Handler(ctx context.Context, onConnect func(context.Context, WSConnection)) http.Handler
}

// WSConnListener represents a WS connection
type WSConnListener struct {
	WSConnection

	ID             string
	eventCallbacks map[string]evtCallback
	onDisconnected func()
	stop           context.CancelFunc

	buffer []byte
}

type evtCallback func([]byte)

func (c *WSConnListener) listen(ctx context.Context) error {
	ctx, c.stop = context.WithCancel(ctx)
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
func (c *WSConnListener) On(event string, callback evtCallback) {
	c.eventCallbacks[event] = callback
}

// OnDisconnected register event callback to closed connections
func (c *WSConnListener) OnDisconnected(fn func()) {
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
func (c *WSConnListener) Emit(message Message) error {
	payload, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	return c.Send(payload)
}

// Close WS connection and stop listening
func (c *WSConnListener) Close() {
	c.stop()
	c.WSConnection.Close()
	go c.onDisconnected()
}

func (c *WSConnListener) readMessage() error {
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

// WSServer manage WS connections
type WSServer struct {
	handler     WSDriver
	onConnected func(c *WSConnListener)

	connections connectionGroup
	sync.RWMutex
}

type connectionGroup map[string]*WSConnListener

// NewWSServer create a new WSServer
func NewWSServer(handler WSDriver) *WSServer {
	wss := &WSServer{handler: handler, onConnected: func(c *WSConnListener) {}}
	wss.connections = make(connectionGroup)
	return wss
}

// OnConnected register event callback to new connections
func (wss *WSServer) OnConnected(fn func(c *WSConnListener)) {
	if fn != nil {
		wss.onConnected = fn
	}
}

// Listen to WS connections
func (wss *WSServer) Listen(ctx context.Context) http.Handler {
	return wss.handler.Handler(ctx, func(ctx context.Context, c WSConnection) {
		conn := wss.Add(c)
		err := withRecover(func() error {
			wss.onConnected(conn)
			defer wss.Remove(conn.ID)
			return conn.listen(ctx)
		})
		if err != nil {
			log.Println("WSServer: read error", err)
		}
	})
}

// Get Conn by session id
func (wss *WSServer) Get(id string) *WSConnListener {
	wss.RLock()
	c := wss.connections[id]
	wss.RUnlock()
	return c
}

// Add Conn for session id
func (wss *WSServer) Add(c WSConnection) *WSConnListener {
	id := uuid.NewV4().String()
	conn := &WSConnListener{c, id, make(map[string]evtCallback), func() {}, func() {}, make([]byte, 512)}
	wss.Lock()
	wss.connections[conn.ID] = conn
	wss.Unlock()
	return conn
}

// Remove Conn by session id
func (wss *WSServer) Remove(id string) {
	if c := wss.Get(id); c != nil {
		c.Close()

		wss.Lock()
		delete(wss.connections, id)
		wss.Unlock()
	}
}

// Emit send payload on eventX to socket id
func (wss *WSServer) Emit(id string, message Message) error {
	if conn := wss.Get(id); conn != nil {
		return conn.Emit(message)
	}
	return ErrWSConnectionNotFound
}

// BroadcastTo ids event message
func (wss *WSServer) BroadcastTo(ids []string, message Message) {
	for _, id := range ids {
		if err := wss.Emit(id, message); err != nil {
			log.Println("error to emit ", message, message, err)
		}
	}
}

// BroadcastFrom connection id event message to all connections
func (wss *WSServer) BroadcastFrom(fromID string, message Message) error {
	wss.RLock()
	connections := wss.connections
	wss.RUnlock()
	for _, c := range connections {
		if fromID == c.ID {
			continue
		}
		if err := c.Emit(message); err != nil {
			return err
		}
	}
	return nil
}

// Broadcast event message to all connections
func (wss *WSServer) Broadcast(message Message) error {
	wss.RLock()
	connections := wss.connections
	wss.RUnlock()
	for _, c := range connections {
		if err := c.Emit(message); err != nil {
			return err
		}
	}
	return nil
}

// Close connection by id
func (wss *WSServer) Close(id string) error {
	if conn := wss.Get(id); conn != nil {
		conn.Close()
		return nil
	}
	return ErrWSConnectionNotFound
}

// CloseAll Conn
func (wss *WSServer) CloseAll() {
	wss.RLock()
	connections := wss.connections
	wss.RUnlock()
	for _, c := range connections {
		c.Close()
	}
}
