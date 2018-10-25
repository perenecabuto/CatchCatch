package main

import (
	"context"
	"log"
	"net/url"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
)

// WebSocket
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
	conn *websocket.Conn
}

var _ WebSocket = &GorillaWebSocket{}

func (ws GorillaWebSocket) NewConnection(url string) (WebSocket, error) {
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "can't dial")
	}
	ws.conn = c
	return &ws, nil
}

func (ws *GorillaWebSocket) OnClose(fn func()) {
	ws.conn.SetCloseHandler(func(code int, text string) error {
		fn()
		return nil
	})
}

func (ws *GorillaWebSocket) Listen(ctx context.Context) chan *WebSocketMessage {
	msgChann := make(chan *WebSocketMessage, 1)
	go func() {
		defer close(msgChann)
		for {
			_, message, err := ws.conn.ReadMessage()
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
	return errors.Cause(ws.Close())
}

// Client
type Client struct {
	ws WebSocket
}

func (c *Client) ConnectAsPlayer(ctx context.Context, addr string) (*Player, error) {
	url := url.URL{Scheme: "ws", Host: addr, Path: "/player"}
	ws, err := c.ws.NewConnection(url.String())
	if err != nil {
		return nil, errors.Wrapf(err, "can't connect to %s", url.String())
	}
	p := NewPlayer(ws)
	go func() {
		err := p.listen(ctx)
		if err != nil {
			log.Printf("player listen error: %+v", err)
		}
	}()
	return p, nil
}

// Player
type EventHandlerFunc func(...interface{}) error
type EventHandler struct {
	core.GameWatcherEvent
	EventHandlerFunc
}

type Player struct {
	ws               WebSocket
	state            game.Player
	eventHandlerChan chan EventHandler
	eventHandlers    map[core.GameWatcherEvent]EventHandlerFunc
}

func NewPlayer(ws WebSocket) *Player {
	return &Player{
		ws:               ws,
		state:            game.Player{},
		eventHandlerChan: make(chan EventHandler, 1),
		eventHandlers:    make(map[core.GameWatcherEvent]EventHandlerFunc),
	}
}

type Rank map[string]int

func (p *Player) listen(ctx context.Context) error {
	chann := p.ws.Listen(ctx)
	p.ws.OnClose(func() {
		log.Println("disconnected")
		close(chann)
	})

	for {
		select {
		case handler := <-p.eventHandlerChan:
			p.eventHandlers[handler.GameWatcherEvent] = handler.EventHandlerFunc
		case msg, ok := <-chann:
			if !ok {
				return nil
			}
			if msg.err != nil {
				return errors.Wrapf(msg.err,
					"error listening player:%s msg", p.state.ID)
			}
			payload := &protobuf.Simple{}
			if err := proto.Unmarshal(msg.data, payload); err != nil {
				return errors.Wrap(err, "can't parse message")
			}
			log.Println("Received event:", payload.GetEventName())

			switch payload.GetEventName() {
			case core.EventPlayerRegistered:
				payload := protobuf.Player{}
				if err := proto.Unmarshal(msg.data, &payload); err != nil {
					return errors.Wrap(err, "can't parse player")
				}
				if p.state.Lat != 0 || p.state.Lon != 0 {
					p.UpdatePlayer(p.state.Lat, p.state.Lon)
				}
			case core.EventPlayerDisconnect:
				return errors.Wrap(p.ws.Close(), "on disconnect")

			case core.GameStarted.String():
				payload := protobuf.GameInfo{}
				if err := proto.Unmarshal(msg.data, &payload); err != nil {
					return errors.Wrap(err, "can't parse game info")
				}
				fn, ok := p.eventHandlers[core.GameStarted]
				if ok {
					fn(payload.GetGame(), payload.GetRole())
				}

			case core.GamePlayerNearToTarget.String():
				payload := protobuf.Distance{}
				if err := proto.Unmarshal(msg.data, &payload); err != nil {
					return errors.Wrap(err, "can't parse game info")
				}
				fn, ok := p.eventHandlers[core.GamePlayerNearToTarget]
				if ok {
					fn(payload.GetDist())
				}

			case core.GamePlayerLose.String():
				payload := protobuf.Simple{}
				if err := proto.Unmarshal(msg.data, &payload); err != nil {
					return errors.Wrap(err, "can't parse game info")
				}
				fn, ok := p.eventHandlers[core.GamePlayerLose]
				if ok {
					fn()
				}

			case core.GamePlayerWin.String():
				payload := protobuf.Distance{}
				if err := proto.Unmarshal(msg.data, &payload); err != nil {
					return errors.Wrap(err, "can't parse game info")
				}
				fn, ok := p.eventHandlers[core.GamePlayerWin]
				if ok {
					fn(payload.GetDist())
				}

			case core.GameFinished.String():
				payload := protobuf.GameRank{}
				if err := proto.Unmarshal(msg.data, &payload); err != nil {
					return errors.Wrap(err, "can't parse game info")
				}
				rank := make(Rank)
				for _, r := range payload.GetPlayersRank() {
					rank[r.GetPlayer()] = int(r.GetPoints())
				}
				fn, ok := p.eventHandlers[core.GameFinished]
				if ok {
					fn(payload.GetGame(), rank)
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (p *Player) UpdatePlayer(lat, lon float64) error {
	p.state.Lat, p.state.Lon = lat, lon
	payload := protobuf.Player{Id: &p.state.ID, Lat: &p.state.Lat, Lon: &p.state.Lon,
		EventName: proto.String(core.EventPlayerUpdate)}
	data, _ := proto.Marshal(&payload)
	err := p.ws.Send(data)
	return errors.Wrapf(err, "can't update player:%+v", p.state.ID)
}

type LatLon [2]float64

func (p *Player) Coords() LatLon {
	return [2]float64{p.state.Lat, p.state.Lon}
}

func (p *Player) Disconnect() error {
	// TODO: stop listener
	return p.ws.Close()
}

func (p *Player) OnGameStarted(fn func(game, role string) error) {
	p.eventHandlerChan <- EventHandler{core.GameStarted,
		func(params ...interface{}) error {
			return fn(params[0].(string), params[1].(string))
		}}
}
func (p *Player) OnGamePlayerNearToTarget(fn func(dist float64) error) {
	p.eventHandlerChan <- EventHandler{core.GamePlayerNearToTarget,
		func(params ...interface{}) error { return fn(params[0].(float64)) }}
}
func (p *Player) OnGamePlayerLose(fn func() error) {
	p.eventHandlerChan <- EventHandler{core.GamePlayerLose,
		func(params ...interface{}) error { return fn() }}
}
func (p *Player) OnGamePlayerWin(fn func(dist float64) error) {
	p.eventHandlerChan <- EventHandler{core.GamePlayerWin,
		func(params ...interface{}) error { return fn(params[0].(float64)) }}
}
func (p *Player) OnGameFinished(fn func(game string, rank Rank) error) {
	p.eventHandlerChan <- EventHandler{core.GameFinished,
		func(params ...interface{}) error { return fn(params[0].(string), params[1].(Rank)) }}
}
