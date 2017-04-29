package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"

	"strings"

	"fmt"

	uuid "github.com/satori/go.uuid"
	websocket "golang.org/x/net/websocket"
)

// Conn represents a websocket connection
type Conn struct {
	ID             string
	conn           *websocket.Conn
	messagebuf     string
	eventCallbacks map[string]evtCallback
}

type evtCallback func(string)

func (c *Conn) listen(ctx context.Context, doneFunc func(error)) {
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

func (c *Conn) On(event string, callback evtCallback) {
	c.eventCallbacks[event] = callback
}

// Emit send payload on eventX to socket id
func (c *Conn) Emit(event string, message interface{}) error {
	payload, err := parsePayload(message)
	if err != nil {
		return err
	}
	log.Println("Send event to", event, payload)
	if _, err := c.conn.Write([]byte(event + "," + payload)); err != nil {
		return err
	}
	return nil
}

func (c *Conn) close() {
	c.conn.Close()
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
		log.Println("message:", data)
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

func (wss *WebSocketServer) OnConnected(fn func(c *Conn)) {
	if fn != nil {
		wss.onConnected = fn
	}
}

// Listen to websocket connections
func (wss *WebSocketServer) Listen(ctx context.Context) websocket.Handler {
	// websocket handler
	return websocket.Handler(func(c *websocket.Conn) {
		defer func() {
			err := c.Close()
			if err != nil {
				log.Println("Error to close ws:", c)
				return
			}
		}()

		conn := wss.Add(c)

		wss.onConnected(conn)
		conn.listen(ctx, func(err error) {
			if err != nil {
				log.Println("WebSocketServer: read error", err)
			}
			wss.Remove(conn.ID)
		})
	})
}

func (wss *WebSocketServer) watchConnections(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(wss.addCH)
			close(wss.delCH)
			return
		case conn := <-wss.addCH:
			wss.connections[conn.ID] = conn
		case id := <-wss.delCH:
			if c, exists := wss.connections[id]; exists {
				c.close()
				delete(wss.connections, id)
			}
		}
	}
}

// Get Conn by session id
func (wss *WebSocketServer) Get(id string) *Conn {
	if c, exists := wss.connections[id]; exists {
		return c
	}
	return nil
}

// Add Conn for session id
func (wss *WebSocketServer) Add(c *websocket.Conn) *Conn {
	id := uuid.NewV4().String()
	conn := &Conn{id, c, "", make(map[string]evtCallback)}
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
