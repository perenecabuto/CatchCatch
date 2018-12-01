package client

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	gws "github.com/gorilla/websocket"
	xws "golang.org/x/net/websocket"
)

type WebSocketMessage struct {
	Data []byte
	Err  error
}

type WebSocket interface {
	NewConnection(url string) (WebSocket, error)
	Listen(ctx context.Context) chan *WebSocketMessage
	Send([]byte) error
	Close() error
	OnClose(func())
}

// GorillaWebSocket ...
type GorillaWebSocket struct {
	conn    *gws.Conn
	onclose func()
}

func NewGorillaWebSocket() WebSocket {
	return &GorillaWebSocket{onclose: func() {}}
}

func (ws GorillaWebSocket) NewConnection(url string) (WebSocket, error) {
	c, _, err := gws.DefaultDialer.Dial(url, nil)
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
			if _, ok := err.(*gws.CloseError); ok {
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
	err := ws.conn.WriteMessage(gws.BinaryMessage, payload)
	return errors.Wrapf(err, "can't write message to socket")
}

func (ws *GorillaWebSocket) Close() error {
	ws.onclose()
	ws.onclose = func() {}
	return errors.Cause(ws.conn.Close())
}

// XNetWebSocket ...
type XNetWebSocket struct {
	conn    *xws.Conn
	onclose func()
}

func NewXNetWebSocket() WebSocket {
	return &XNetWebSocket{onclose: func() {}}
}

func (ws XNetWebSocket) NewConnection(url_ string) (WebSocket, error) {
	conf, err := xws.NewConfig(url_, url_)
	if err != nil {
		return nil, errors.Cause(err)
	}
	c, err := xws.DialConfig(conf)
	if err != nil {
		return nil, errors.Wrap(err, "can't dial")
	}
	ws.conn = c
	return &ws, nil
}

func (ws *XNetWebSocket) OnClose(fn func()) {
	if fn == nil {
		return
	}
	ws.onclose = fn
}

func (ws *XNetWebSocket) Listen(ctx context.Context) chan *WebSocketMessage {
	msgChann := make(chan *WebSocketMessage, 1)
	go func() {
		defer close(msgChann)
		buff := make([]byte, 4096)
		for {
			n, err := ws.conn.Read(buff)
			if err != nil {
				ws.Close()
				return
			}
			msg := buff[:n]
			if strings.TrimSpace(string(msg)) == "" {
				continue
			}
			payload := &WebSocketMessage{msg, errors.Wrap(err, "can't read websocket")}
			select {
			case msgChann <- payload:
			case <-ctx.Done():
				return
			}
		}
	}()
	return msgChann
}

func (ws *XNetWebSocket) Send(payload []byte) error {
	_, err := ws.conn.Write(payload)
	return errors.Wrapf(err, "can't write message to socket")
}

func (ws *XNetWebSocket) Close() error {
	ws.onclose()
	ws.onclose = func() {}
	return errors.Cause(ws.conn.Close())
}
