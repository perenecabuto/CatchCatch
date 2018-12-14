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
	Cookies() []*http.Cookie
	Read(*[]byte) (int, error)
	Send(payload []byte) error
	Close() error
}

// WSConnectionHandler represents a WS connection
type WSConnectionHandler struct {
	ID string

	conn           WSConnection
	eventCallbacks map[string]WSEventCallback
	buffer         []byte
	onDisconnected func()
	stop           context.CancelFunc
}

// NewWSConnectionHandler creates a new WSConnectionHandler
func NewWSConnectionHandler(c WSConnection, id string) *WSConnectionHandler {
	return &WSConnectionHandler{id, c, make(map[string]WSEventCallback), make([]byte, 512), func() {}, func() {}}
}

// WSEventCallback is called when a event happens
type WSEventCallback func([]byte)

func (ch *WSConnectionHandler) listen(ctx context.Context) error {
	ctx, ch.stop = context.WithCancel(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := ch.readMessage(); err != nil {
				return err
			}
		}
	}
}

// On this connection event trigger callback with its message
func (ch *WSConnectionHandler) On(event string, callback WSEventCallback) {
	ch.eventCallbacks[event] = callback
}

// OnDisconnected register event callback to closed connections
func (ch *WSConnectionHandler) OnDisconnected(fn func()) {
	if fn != nil {
		ch.onDisconnected = fn
	}
}

// Message represent protobuf message with event name
type Message interface {
	proto.Message
	GetEventName() string
}

// Emit send payload on eventX to socket id
func (ch *WSConnectionHandler) Emit(message Message) error {
	payload, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	return ch.conn.Send(payload)
}

// Close WS connection and stop listening
func (ch *WSConnectionHandler) Close() {
	ch.stop()
	ch.conn.Close()
	go ch.onDisconnected()
}

func (ch *WSConnectionHandler) readMessage() error {
	length, err := ch.conn.Read(&ch.buffer)
	if err != nil {
		return err
	}
	if length == 0 {
		return nil
	}
	msg := &protobuf.Simple{}
	if err := proto.Unmarshal(ch.buffer[:length], msg); err != nil {
		return fmt.Errorf("readMessage(unmarshall): %s %s", err.Error(), ch.buffer[:length])
	}
	if len(msg.String()) == 0 {
		log.Println("message error:", msg)
		return fmt.Errorf("Invalid payload: %s", ch.buffer)
	}
	cb, exists := ch.eventCallbacks[msg.GetEventName()]
	if !exists {
		return fmt.Errorf("No callback found for: %v", msg)
	}
	return execfunc.WithRecover(func() error {
		cb(ch.buffer)
		return nil
	})
}
