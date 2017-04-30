package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	uuid "github.com/satori/go.uuid"
	websocket "golang.org/x/net/websocket"
)

// Conn represents a websocket connection
type Conn struct {
	ID   string
	conn *websocket.Conn

	messagebuf     string
	eventCallbacks map[string]evtCallback
	onDisconnected func()
	cancelFN       context.CancelFunc
}

func NewConn(conn *websocket.Conn) *Conn {
	id := uuid.NewV4().String()
	return &Conn{id, conn, "", make(map[string]evtCallback), func() {}, func() {}}
}

type evtCallback func(string)

func (c *Conn) listen(ctx context.Context, doneFunc func(error)) {
	ctx, c.cancelFN = context.WithCancel(ctx)
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
	_, err = c.conn.Write([]byte(event + "," + payload))
	return err
}

func (c *Conn) close() {
	c.cancelFN()
	c.conn.Close()
	c.onDisconnected()
}

func (c *Conn) readMessage() error {
	if err := websocket.Message.Receive(c.conn, &c.messagebuf); err != nil {
		return err
	}
	data := strings.SplitN(c.messagebuf, ",", 2)
	if len(data) == 0 {
		log.Println("message error:", c.messagebuf)
		return errors.New("Invalid payload: " + c.messagebuf)
	}
	if cb, exists := c.eventCallbacks[data[0]]; exists {
		cb(data[1])
		return nil

	}
	return fmt.Errorf("No callback found: %v", data)
}

// WebSocketServer manage websocket connections
type WebSocketServer struct {
	connections map[string]*Conn
	addCH       chan *Conn
	delCH       chan string

	onConnected func(c *Conn)
}

// NewWebSocketServer create a new WebSocketServer
func NewWebSocketServer(ctx context.Context) *WebSocketServer {
	server := &WebSocketServer{make(map[string]*Conn), make(chan *Conn), make(chan string), func(c *Conn) {}}
	go server.watchConnections(ctx)
	return server
}

// OnConnected register event callback to new connections
func (wss *WebSocketServer) OnConnected(fn func(c *Conn)) {
	if fn != nil {
		wss.onConnected = fn
	}
}

// Listen to websocket connections
func (wss *WebSocketServer) Listen(ctx context.Context) websocket.Handler {
	// websocket handler
	return websocket.Handler(func(c *websocket.Conn) {
		conn := wss.Add(c)
		defer wss.Remove(conn.ID)
		wss.onConnected(conn)
		conn.listen(ctx, func(err error) {
			if err != nil {
				log.Println("WebSocketServer: read error", err)
			}
		})
	})
}

func (wss *WebSocketServer) watchConnections(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Stop watch connections")
			close(wss.addCH)
			close(wss.delCH)
			return
		case conn := <-wss.addCH:
			wss.connections[conn.ID] = conn
		case id := <-wss.delCH:
			log.Println("Delete conn", id)
			if c, exists := wss.connections[id]; exists {
				delete(wss.connections, id)
				c.close()
			}
		}
	}
	log.Println("Stop watch connections!!!!")

}

// Get Conn by session id
func (wss *WebSocketServer) Get(id string) *Conn {
	return wss.connections[id]
}

// Add Conn for session id
func (wss *WebSocketServer) Add(c *websocket.Conn) *Conn {
	conn := NewConn(c)
	wss.addCH <- conn
	return conn
}

// Remove Conn by session id
func (wss *WebSocketServer) Remove(id string) {
	wss.delCH <- id
}

// Emit send payload on eventX to socket id
func (wss *WebSocketServer) Emit(id, event string, message interface{}) error {
	if conn := wss.Get(id); conn != nil {
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
		return strconv.Quote(msg.(string)), nil
	default:
		jPayload, err := json.Marshal(msg)
		if err != nil {
			return "", err
		}
		return string(jPayload), nil
	}
}
