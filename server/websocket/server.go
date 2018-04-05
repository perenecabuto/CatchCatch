package websocket

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/golang/protobuf/proto"

	"github.com/perenecabuto/CatchCatch/server/execfunc"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
)

var (
	// ErrWSConnectionNotFound is returned when connection id is not registred on this server
	ErrWSConnectionNotFound = errors.New("connection not found")
)

// WSDriver is an interface for WS communication
type WSDriver interface {
	HTTPHandler(ctx context.Context, onConnect func(context.Context, WSConnection)) http.Handler
}

// WSServer manage WS connections
type WSServer struct {
	driver       WSDriver
	OnConnection func(c *WSConnListener)

	connections connectionGroup
	sync.RWMutex
}

type connectionGroup map[string]*WSConnListener

// NewWSServer create a new WSServer
func NewWSServer(driver WSDriver) *WSServer {
	wss := &WSServer{driver: driver, OnConnection: func(c *WSConnListener) {}}
	wss.connections = make(connectionGroup)
	return wss
}

// WSEventHandler is responsible to handle WSConnListener events
type WSEventHandler interface {
	OnConnection(c *WSConnListener)
}

// SetEventHandler set the event handler to new connections
func (wss *WSServer) SetEventHandler(h WSEventHandler) {
	if h != nil {
		wss.OnConnection = h.OnConnection
	}
}

// Listen to WS connections
func (wss *WSServer) Listen(ctx context.Context) http.Handler {
	return wss.driver.HTTPHandler(ctx, func(ctx context.Context, c WSConnection) {
		conn := wss.Add(c)
		err := execfunc.WithRecover(func() error {
			conn.Emit(&protobuf.Simple{EventName: proto.String("connect"), Id: &conn.ID})
			wss.OnConnection(conn)
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
	conn := NewWSConnListener(c)
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

// EmitTo ids event message
func (wss *WSServer) EmitTo(ids []string, message Message) {
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
