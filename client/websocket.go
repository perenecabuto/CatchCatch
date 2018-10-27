package client

import (
	"context"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

type WebSocketMessage struct {
	data []byte
	err  error
}

type WebSocket interface {
	NewConnection(url string) (WebSocket, error)
	Listen(ctx context.Context) chan *WebSocketMessage
	Send([]byte) error
	Close() error
	OnClose(func())
}

type GorillaWebSocket struct {
	conn    *websocket.Conn
	onclose func()
}

func NewGorillaWebSocket() WebSocket {
	return &GorillaWebSocket{onclose: func() {}}
}

func (ws GorillaWebSocket) NewConnection(url string) (WebSocket, error) {
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "can't dial")
	}
	ws.conn = c
	return &ws, nil
}

func (ws *GorillaWebSocket) OnClose(fn func()) {
	if fn == nil {
		return
	}
	ws.onclose = fn
	ws.conn.SetCloseHandler(func(int, string) error {
		ws.onclose()
		return nil
	})
}

func (ws *GorillaWebSocket) Listen(ctx context.Context) chan *WebSocketMessage {
	msgChann := make(chan *WebSocketMessage, 1)
	go func() {
		defer close(msgChann)
		for {
			_, message, err := ws.conn.ReadMessage()
			if _, ok := err.(*websocket.CloseError); ok {
				ws.Close()
				return
			}
			payload := &WebSocketMessage{message, errors.Wrap(err, "can't read websocket")}
			select {
			case msgChann <- payload:
			case <-ctx.Done():
				return
			}
		}
	}()
	return msgChann
}

func (ws *GorillaWebSocket) Send(payload []byte) error {
	err := ws.conn.WriteMessage(websocket.BinaryMessage, payload)
	return errors.Wrapf(err, "can't write message to socket")
}

func (ws *GorillaWebSocket) Close() error {
	ws.onclose()
	ws.onclose = func() {}
	return errors.Cause(ws.conn.Close())
}
