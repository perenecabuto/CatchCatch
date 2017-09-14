package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/golang/protobuf/proto"

	"github.com/perenecabuto/CatchCatch/catchcatch-server/protobuf"
)

// EventHandler handle websocket events
type EventHandler struct {
	server  *WebSocketServer
	service PlayerLocationService
	games   *GameWatcher
}

// NewEventHandler EventHandler builder
func NewEventHandler(server *WebSocketServer, service PlayerLocationService, gw *GameWatcher) *EventHandler {
	handler := &EventHandler{server, service, gw}
	server.OnConnected(handler.onConnection)
	return handler
}

// Listen to websocket connections
func (h *EventHandler) Listen(ctx context.Context) http.Handler {
	return h.server.Listen(ctx)
}

// Event handlers

func (h *EventHandler) onConnection(c *Conn) {
	player, err := h.newPlayer(c)
	if err != nil {
		log.Println("error to create player", err)
		c.close()
		return
	}
	log.Println("new player connected", player)
	go h.sendPlayerList(c)

	c.On("player:request-games", h.onPlayerRequestGames(player, c))
	c.On("player:request-remotes", h.onPlayerRequestRemotes(c))
	c.On("player:update", h.onPlayerUpdate(player, c))
	c.OnDisconnected(h.onPlayerDisconnect(player))

	c.On("admin:disconnect", h.onDisconnectByID())
	c.On("admin:feature:add", h.onAddFeature())
	c.On("admin:feature:request-list", h.onRequestFeatures(c))
	c.On("admin:clear", h.onClear())
}

// Player events

func (h *EventHandler) onPlayerDisconnect(player *Player) func() {
	return func() {
		log.Println("player:disconnect", player.ID)
		h.server.Broadcast(&protobuf.Player{EventName: proto.String("remote-player:destroy"),
			Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
		h.service.Remove(player)
	}
}

func (h *EventHandler) onPlayerUpdate(player *Player, c *Conn) func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Player{}
		proto.Unmarshal(buf, msg)

		player.Lat, player.Lon = *msg.Lat, *msg.Lon
		h.service.Update(player)

		c.Emit(&protobuf.Player{EventName: proto.String("player:updated"),
			Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
		h.server.Broadcast(&protobuf.Player{EventName: proto.String("remote-player:updated"),
			Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
	}
}

func (h *EventHandler) onPlayerRequestRemotes(so *Conn) func([]byte) {
	return func([]byte) {
		h.sendPlayerList(so)
	}
}

func (h *EventHandler) onPlayerRequestGames(player *Player, c *Conn) func([]byte) {
	return func([]byte) {
		go func() {
			games, err := h.service.FeaturesAround("geofences", player.Point())
			if err != nil {
				log.Println("Error to request games:", err)
				return
			}
			event := proto.String("game:around")
			for _, f := range games {
				err := c.Emit(&protobuf.Feature{EventName: event, Id: &f.ID, Group: &f.Group, Coords: &f.Coordinates})
				if err != nil {
					log.Println("Error to emit", *event, player)
				}
			}
		}()
	}
}

// Admin events

func (h *EventHandler) onDisconnectByID() func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Simple{}
		proto.Unmarshal(buf, msg)
		log.Println("admin:disconnect", msg.GetId())
		h.server.Remove(msg.GetId())
	}
}

func (h *EventHandler) onClear() func([]byte) {
	return func([]byte) {
		h.games.Clear()
		h.service.Clear()
		h.server.CloseAll()
	}
}

// Map events

func (h *EventHandler) onAddFeature() func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Feature{}
		proto.Unmarshal(buf, msg)

		f, err := h.service.AddFeature(msg.GetGroup(), msg.GetId(), msg.GetCoords())
		if err != nil {
			log.Println("Error to create feature:", err)
			return
		}
		h.server.Broadcast(&protobuf.Feature{EventName: proto.String("admin:feature:added"), Id: &f.ID, Group: &f.Group, Coords: &f.Coordinates})
	}
}

func (h *EventHandler) onRequestFeatures(c *Conn) func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Feature{}
		proto.Unmarshal(buf, msg)

		features, err := h.service.Features(msg.GetGroup())
		if err != nil {
			log.Println("Error on sendFeatures:", err)

		}
		event := "admin:feature:added"
		for _, f := range features {
			c.Emit(&protobuf.Feature{EventName: &event, Id: &f.ID, Group: &f.Group, Coords: &f.Coordinates})
		}
	}
}

// Actions

func (h *EventHandler) newPlayer(c *Conn) (player *Player, err error) {
	player = &Player{c.ID, 0, 0}
	if err := h.service.Register(player); err != nil {
		return nil, errors.New("could not register: " + err.Error())
	}
	c.Emit(&protobuf.Player{EventName: proto.String("player:registered"), Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
	h.server.Broadcast(&protobuf.Player{EventName: proto.String("remote-player:new"), Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
	return player, nil
}

func (h *EventHandler) sendPlayerList(c *Conn) error {
	return withRecover(func() error {
		players, err := h.service.Players()
		if err != nil {
			return errors.New("player:request-remotes event error: " + err.Error())
		}
		event := "remote-player:new"
		for _, p := range players {
			if p == nil {
				continue
			}
			err := c.Emit(&protobuf.Player{EventName: &event, Id: &p.ID, Lon: &p.Lon, Lat: &p.Lat})
			if err != nil {
				return errors.New("player:request-remotes event error: " + err.Error())
			}
		}
		return nil
	})
}
