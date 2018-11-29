// +build js,wasm

package main

import (
	"context"
	"errors"
	"log"
	"syscall/js"
	"time"

	"github.com/perenecabuto/CatchCatch/client"
)

type WASMWebSocket struct {
	browserWS js.Value
	messages  chan *client.WebSocketMessage
}

var _ client.WebSocket = (*WASMWebSocket)(nil)

func NewWASMWebSocket() *WASMWebSocket {
	return &WASMWebSocket{
		messages: make(chan *client.WebSocketMessage),
	}
}

func (ws WASMWebSocket) NewConnection(url string) (client.WebSocket, error) {
	ws.browserWS = js.Global().Get("WebSocket").New(url)
	ws.browserWS.Set("binaryType", "arraybuffer")

	ws.browserWS.Set("onconnect", callbackFunc(func(values []js.Value) {
		log.Println("connected as ", values)
		readyEvt := js.Global().Get("Event").New("catchcatch:player:connected")
		js.Global().Get("document").Call("dispatchEvent", readyEvt)
	}))

	ws.browserWS.Set("onmessage", callbackFunc(func(values []js.Value) {
		if len(values) != 1 {
			logError("onmessage: received empty event data")
			return
		}
		encoded := values[0].Get("data")
		data := js.Global().Get("Uint8Array").New(encoded)
		if data.Length() == 0 {
			logError("onmessage: message data is empty")
			return
		}
		converted := make([]byte, data.Length())
		for i := 0; i < data.Length(); i++ {
			converted[i] = byte(data.Index(i).Int())
		}
		log.Printf("SUCCESS onmessage: %+v", string(converted))
		msg := &client.WebSocketMessage{Data: converted}
		select {
		case <-time.NewTimer(time.Second).C:
		case ws.messages <- msg:
		}
	}))
	ws.browserWS.Set("onerror", callbackFunc(func(values []js.Value) {
		err := errors.New(values[0].String())
		msg := &client.WebSocketMessage{Err: err}
		select {
		case <-time.NewTimer(time.Second).C:
		case ws.messages <- msg:
		}
	}))
	return &ws, nil
}

func (ws *WASMWebSocket) Listen(ctx context.Context) chan *client.WebSocketMessage {
	return ws.messages
}

func (ws *WASMWebSocket) Send(msg []byte) error {
	ws.browserWS.Call("send", js.TypedArrayOf(msg))
	return nil
}

func (ws *WASMWebSocket) Close() error {
	ws.browserWS.Call("close")
	return nil
}

func (ws *WASMWebSocket) OnClose(cb func()) {
	ws.browserWS.Set("onclose", callbackFunc(func(values []js.Value) {
		cb()
	}))
}
