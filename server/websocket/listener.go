package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/golang/protobuf/proto"

	"github.com/perenecabuto/CatchCatch/server/execfunc"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
)

// WSConnection is an interface for WS communication
type WSConnection interface {
	Read(*[]byte) (int, error)
	Send(payload []byte) error
	Close() error
}

// WSConnectionHandler represents a WS connection
type WSConnectionHandler struct {
	WSConnection

	ID             string
	eventCallbacks map[string]WSEventCallback
	onDisconnected func()
	stop           context.CancelFunc

	buffer []byte
}

// NewWSConnectionHandler creates a new WSConnectionHandler
func NewWSConnectionHandler(c WSConnection, id string) *WSConnectionHandler {
	return &WSConnectionHandler{c, id, make(map[string]WSEventCallback), func() {}, func() {}, make([]byte, 512)}
}

// WSEventCallback is called when a event happens
type WSEventCallback func([]byte)

func (c *WSConnectionHandler) listen(ctx context.Context) error {
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
func (c *WSConnectionHandler) On(event string, callback WSEventCallback) {
	c.eventCallbacks[event] = callback
}

// OnDisconnected register event callback to closed connections
func (c *WSConnectionHandler) OnDisconnected(fn func()) {
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
func (c *WSConnectionHandler) Emit(message Message) error {
	payload, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	return c.Send(payload)
}

// Close WS connection and stop listening
func (c *WSConnectionHandler) Close() {
	c.stop()
	c.WSConnection.Close()
	go c.onDisconnected()
}

func (c *WSConnectionHandler) readMessage() error {
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
	return execfunc.WithRecover(func() error {
		cb(c.buffer)
		return nil
	})
}
