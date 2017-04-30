package main

import (
	"context"
	"errors"
	"log"

	gjson "github.com/tidwall/gjson"
	websocket "golang.org/x/net/websocket"
)

// EventHandler handle socket.io events
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

// Listen listen to websocket connections
func (h *EventHandler) Listen(ctx context.Context) websocket.Handler {
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
		h.server.Broadcast("remote-player:destroy", player)
		h.service.Remove(player)
	}
}

func (h *EventHandler) onPlayerUpdate(player *Player, c *Conn) func(string) {
	return func(msg string) {
		coords := gjson.GetMany(msg, "lat", "lon")
		player.Lat, player.Lon = coords[0].Float(), coords[1].Float()
		c.Emit("player:updated", player)
		h.server.Broadcast("remote-player:updated", player)
		h.service.Update(player)
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
			}
			log.Println("game:around", games)
			if err = c.Emit("game:around", games); err != nil {
				log.Println("Error to emit", "game:around", player)
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
		feature, err := h.service.AddFeature(group, name, geojson)
		if err != nil {
			log.Println("Error to create feature:", err)
		}
		log.Println("Added feature", feature)
		h.server.Broadcast("admin:feature:added", feature)
	}
}

func (h *EventHandler) onRequestFeatures(c *Conn) func(string) {
	return func(group string) {
		features, err := h.service.Features(group)
		if err != nil {
			log.Println("Error on sendFeatures:", err)

		}
		c.Emit("admin:feature:list", features)
	}
}

// Actions

func (h *EventHandler) newPlayer(c *Conn) (player *Player, err error) {
	player = &Player{c.ID, 0, 0}
	if err := h.service.Register(player); err != nil {
		return nil, errors.New("could not register: " + err.Error())
	}

	c.Emit("player:registered", player)
	h.server.Broadcast("remote-player:new", player)
	return player, nil
}

func (h *EventHandler) sendPlayerList(c *Conn) {
	if players, err := h.service.Players(); err != nil {
		log.Println("player:request-remotes event error: " + err.Error())
	} else if err := c.Emit("remote-player:list", players); err != nil {
		log.Println("player:request-remotes event error: " + err.Error())
	}
}
