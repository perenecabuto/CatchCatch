package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	protobuf "./protobuf"
	"github.com/golang/protobuf/proto"
	gjson "github.com/tidwall/gjson"
)

// EventHandler handle websocket events
type EventHandler struct {
	server  *WebSocketServer
	service *PlayerLocationService
	games   *GameWatcher
}

// NewEventHandler EventHandler builder
func NewEventHandler(server *WebSocketServer, service *PlayerLocationService, gw *GameWatcher) *EventHandler {
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

func (h *EventHandler) onPlayerUpdate(player *Player, c *Conn) func(string) {
	return func(msg string) {
		coords := gjson.GetMany(msg, "lat", "lon")
		player.Lat, player.Lon = coords[0].Float(), coords[1].Float()
		h.service.Update(player)

		c.Emit(&protobuf.Player{EventName: proto.String("player:updated"),
			Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
		h.server.Broadcast(&protobuf.Player{EventName: proto.String("remote-player:updated"),
			Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
	}
}

func (h *EventHandler) onPlayerRequestRemotes(so *Conn) func(string) {
	return func(string) {
		h.sendPlayerList(so)
	}
}

func (h *EventHandler) onPlayerRequestGames(player *Player, c *Conn) func(string) {
	return func(string) {
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

func (h *EventHandler) onDisconnectByID() func(string) {
	return func(id string) {
		log.Println("admin:disconnect", id)
		h.server.Remove(id)
	}
}

func (h *EventHandler) onClear() func(string) {
	return func(string) {
		h.games.Clear()
		h.service.client.FlushDb()
		h.server.CloseAll()
	}
}

// Map events

func (h *EventHandler) onAddFeature() func(string) {
	return func(msg string) {
		data := gjson.GetMany(msg, "group", "name", "geojson")
		group, name, geojson := data[0].String(), data[1].String(), data[2].String()
		f, err := h.service.AddFeature(group, name, geojson)
		if err != nil {
			log.Println("Error to create feature:", err)
		}
		h.server.Broadcast(&protobuf.Feature{EventName: proto.String("admin:feature:added"), Id: &f.ID, Group: &f.Group, Coords: &f.Coordinates})
	}
}

func (h *EventHandler) onRequestFeatures(c *Conn) func(string) {
	return func(group string) {
		features, err := h.service.Features(group)
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

func (h *EventHandler) sendPlayerList(c *Conn) {
	players, err := h.service.Players()
	if err != nil {
		log.Println("player:request-remotes event error: " + err.Error())
		return
	}
	event := "remote-player:new"
	for _, p := range players {
		err := c.Emit(&protobuf.Player{EventName: &event, Id: &p.ID, Lon: &p.Lon, Lat: &p.Lat})
		if err != nil {
			log.Println("player:request-remotes event error: " + err.Error())
		}
	}
}
