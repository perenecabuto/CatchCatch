package client

import (
	"context"
	"log"
	"net/url"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
)

type Client struct {
	ws WebSocket
}

func New(ws WebSocket) *Client {
	return &Client{ws}
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
	core.GameWorkerEvent
	EventHandlerFunc
}

type Player struct {
	ws               WebSocket
	state            game.Player
	eventHandlerChan chan EventHandler
	eventHandlers    map[core.GameWorkerEvent]EventHandlerFunc
}

func NewPlayer(ws WebSocket) *Player {
	return &Player{
		ws:               ws,
		state:            game.Player{},
		eventHandlerChan: make(chan EventHandler, 1),
		eventHandlers:    make(map[core.GameWorkerEvent]EventHandlerFunc),
	}
}

type Rank map[string]int

func (p *Player) listen(ctx context.Context) error {
	defer func() {
		log.Println("disconnected")
		p.Disconnect()
		fn, ok := p.eventHandlers[core.EventPlayerDisconnect]
		if ok {
			fn()
		}
	}()

	ctx, cancel := context.WithCancel(ctx)
	p.ws.OnClose(func() {
		cancel()
	})

	pingInterval := time.Second * core.PlayerExpirationInSec / 2
	ping := time.NewTimer(pingInterval)
	defer ping.Stop()

	chann := p.ws.Listen(ctx)
	for {
		select {
		case handler := <-p.eventHandlerChan:
			p.eventHandlers[handler.GameWorkerEvent] = handler.EventHandlerFunc
		case <-ping.C:
			if err := p.Ping(); err != nil {
				return errors.Cause(err)
			}
			ping.Reset(pingInterval)

		case msg, ok := <-chann:
			if !ok {
				return nil
			}
			if msg.Err != nil {
				return errors.Wrapf(msg.Err,
					"error listening player:%s msg", p.state.ID)
			}
			payload := &protobuf.Simple{}
			if err := proto.Unmarshal(msg.Data, payload); err != nil {
				return errors.Wrap(err, "can't parse message")
			}
			log.Println("Received event:", payload.GetEventName())

			switch payload.GetEventName() {
			case core.EventPlayerRegistered:
				payload := protobuf.Player{}
				if err := proto.Unmarshal(msg.Data, &payload); err != nil {
					return errors.Wrap(err, "can't parse player")
				}
				p.state.ID = payload.Id
				fn, ok := p.eventHandlers[core.EventPlayerRegistered]
				if ok {
					fn(p.state)
				}

			case core.GameStarted.String():
				payload := protobuf.GameInfo{}
				if err := proto.Unmarshal(msg.Data, &payload); err != nil {
					return errors.Wrap(err, "can't parse game info")
				}
				fn, ok := p.eventHandlers[core.GameStarted]
				if ok {
					fn(payload.GetGame(), payload.GetRole())
				}

			case core.GamePlayerNearToTarget.String():
				payload := protobuf.Distance{}
				if err := proto.Unmarshal(msg.Data, &payload); err != nil {
					return errors.Wrap(err, "can't parse game info")
				}
				fn, ok := p.eventHandlers[core.GamePlayerNearToTarget]
				if ok {
					fn(payload.GetDist())
				}

			case core.GamePlayerLose.String():
				payload := protobuf.Simple{}
				if err := proto.Unmarshal(msg.Data, &payload); err != nil {
					return errors.Wrap(err, "can't parse game info")
				}
				fn, ok := p.eventHandlers[core.GamePlayerLose]
				if ok {
					fn()
				}

			case core.GamePlayerWin.String():
				payload := protobuf.Distance{}
				if err := proto.Unmarshal(msg.Data, &payload); err != nil {
					return errors.Wrap(err, "can't parse game info")
				}
				fn, ok := p.eventHandlers[core.GamePlayerWin]
				if ok {
					fn(payload.GetDist())
				}

			case core.GameFinished.String():
				payload := protobuf.GameRank{}
				if err := proto.Unmarshal(msg.Data, &payload); err != nil {
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

func (p *Player) Ping() error {
	payload := protobuf.Simple{Id: "ping", EventName: core.EventPlayerPing}
	data, _ := proto.Marshal(&payload)
	err := p.ws.Send(data)
	return errors.Wrapf(err, "can't ping:%+v", p.state.ID)
}

func (p *Player) UpdatePlayer(lat, lon float64) error {
	p.state.Lat, p.state.Lon = lat, lon
	payload := protobuf.Player{Id: p.state.ID, Lat: p.state.Lat, Lon: p.state.Lon,
		EventName: core.EventPlayerUpdate}
	data, _ := proto.Marshal(&payload)
	err := p.ws.Send(data)
	return errors.Wrapf(err, "can't update player:%+v", p.state.ID)
}

type LatLon [2]float64

func (p *Player) Coords() LatLon {
	return [2]float64{p.state.Lat, p.state.Lon}
}

func (p *Player) Disconnect() error {
	return p.ws.Close()
}

func (p *Player) OnRegistered(fn func(player game.Player) error) {
	p.eventHandlerChan <- EventHandler{core.EventPlayerRegistered,
		func(params ...interface{}) error {
			return fn(params[0].(game.Player))
		}}
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
func (p *Player) OnDisconnect(fn func() error) {
	p.eventHandlerChan <- EventHandler{core.EventPlayerDisconnect,
		func(params ...interface{}) error { return fn() }}
}
