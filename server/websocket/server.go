package websocket

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/golang/protobuf/proto"

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
	driver  WSDriver
	handler WSEventHandler

	connections connectionGroup
	sync.RWMutex
}

type connectionGroup map[string]*WSConnectionHandler

// NewWSServer create a new WSServer
func NewWSServer(driver WSDriver, handler WSEventHandler) *WSServer {
	wss := &WSServer{driver: driver, handler: handler}
	wss.connections = make(connectionGroup)
	return wss
}

// WSEventHandler is responsible to handle WSConnectionHandler events
type WSEventHandler interface {
	OnStart(context.Context, *WSServer) error
	OnConnection(context.Context, *WSConnectionHandler) error
}

// Listen to WS connections
func (wss *WSServer) Listen(ctx context.Context) http.Handler {
	err := wss.handler.OnStart(ctx, wss)
	if err != nil {
		// TODO: return this error
		log.Panic(err)
		return nil
	}

	return wss.driver.HTTPHandler(ctx, func(connctx context.Context, c WSConnection) {
		conn := wss.Add(c)
		defer wss.Remove(conn.ID)
		err := conn.Emit(&protobuf.Simple{EventName: proto.String("connect"), Id: &conn.ID})
		if err != nil {
			log.Println("[WSServer] Listen: error to notify connect event:", err)
			return
		}
		err = wss.handler.OnConnection(connctx, conn)
		if err != nil {
			log.Println("[WSServer] Listen: handler.OnConnection error:", err)
			return
		}
		err = conn.listen(connctx)
		if err != nil {
			log.Println("[WSServer] Listen: conn.listen error:", err)
		}
	})
}

// Get Conn by session id
func (wss *WSServer) Get(id string) *WSConnectionHandler {
	wss.RLock()
	c := wss.connections[id]
	wss.RUnlock()
	return c
}

// Add Conn for session id
func (wss *WSServer) Add(c WSConnection) *WSConnectionHandler {
	conn := NewWSConnectionHandler(c)
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
