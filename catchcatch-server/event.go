package main

import (
	"errors"
	"log"
	"net/http"

	io "github.com/googollee/go-socket.io"
	gjson "github.com/tidwall/gjson"
)

// EventHandler handle socket.io events
type EventHandler struct {
	server   *io.Server
	service  *PlayerLocationService
	sessions *SessionManager
}

// NewEventHandler EventHandler builder
func NewEventHandler(server *io.Server, service *PlayerLocationService, sessions *SessionManager) *EventHandler {
	server.SetSessionManager(sessions)
	handler := &EventHandler{server, service, sessions}
	server.On("connection", handler.onConnection)
	return handler
}

func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.server.ServeHTTP(w, r)
}

// Event handlers

func (h *EventHandler) onConnection(so io.Socket) {
	channel := "main"
	player, err := h.newPlayer(so, channel)
	if err != nil {
		log.Println("error to create player", err)
		h.sessions.Get(so.Id()).Close()
		return
	}
	log.Println("new player connected", player)
	go h.sendPlayerList(so)

	so.On("player:request-games", h.onPlayerRequestGames(player, so))
	so.On("player:request-remotes", h.onPlayerRequestRemotes(so))
	so.On("player:update", h.onPlayerUpdate(player, channel, so))
	so.On("disconnection", h.onPlayerDisconnect(player, channel))

	so.On("admin:disconnect", h.onDisconnectByID(channel))
	so.On("admin:feature:add", h.onAddFeature(channel))
	so.On("admin:feature:request-list", h.onRequestFeatures(so))
	so.On("admin:clear", h.onClear())
}

// Player events

func (h *EventHandler) onPlayerDisconnect(player *Player, channel string) func(string) {
	return func(string) {
		log.Println("player:disconnect", player.ID)
		h.server.BroadcastTo(channel, "remote-player:destroy", player)
		h.service.Remove(player)
		log.Println("--> diconnected", player)
	}
}

func (h *EventHandler) onPlayerUpdate(player *Player, channel string, so io.Socket) func(string) {
	return func(msg string) {
		coords := gjson.GetMany(msg, "lat", "lon")
		player.Lat, player.Lon = coords[0].Float(), coords[1].Float()
		so.Emit("player:updated", player)
		so.BroadcastTo(channel, "remote-player:updated", player)
		h.service.Update(player)
	}
}

func (h *EventHandler) onPlayerRequestRemotes(so io.Socket) func(string) {
	return func(string) {
		h.sendPlayerList(so)
	}
}

func (h *EventHandler) onPlayerRequestGames(player *Player, so io.Socket) func(string) {
	return func(string) {
		games, err := h.service.FeaturesAround("geofences", player.Point())
		if err != nil {
			log.Println("Error to request games:", err)
		}
		log.Println("game:around", games)
		if err = so.Emit("game:around", games); err != nil {
			log.Println("Error to emit", "game:around", player)
		}
	}
}

// Admin events

func (h *EventHandler) onDisconnectByID(channel string) func(string) {
	return func(id string) {
		log.Println("admin:disconnect", id)
		callback := h.onPlayerDisconnect(&Player{ID: id}, channel)
		callback("")
	}
}

func (h *EventHandler) onClear() func(string) {
	return func(string) {
		h.service.client.FlushDb()
		h.sessions.CloseAll()
	}
}

// Map events

func (h *EventHandler) onAddFeature(channel string) func(group, name, geojson string) {
	return func(group, name, geojson string) {
		feature, err := h.service.AddFeature(group, name, geojson)
		if err != nil {
			log.Println("Error to create feature:", err)
		}
		log.Println("Added feature", feature)
		h.server.BroadcastTo(channel, "admin:feature:added", feature)
	}
}

func (h *EventHandler) onRequestFeatures(so io.Socket) func(string) {
	return func(group string) {
		features, err := h.service.Features(group)
		if err != nil {
			log.Println("Error on sendFeatures:", err)

		}
		so.Emit("admin:feature:list", features)
	}
}

// Actions

func (h *EventHandler) newPlayer(so io.Socket, channel string) (player *Player, err error) {
	player = &Player{so.Id(), 0, 0}
	if err := h.service.Register(player); err != nil {
		return nil, errors.New("could not register: " + err.Error())
	}

	so.Join(channel)
	so.Emit("player:registered", player)
	so.BroadcastTo(channel, "remote-player:new", player)
	return player, nil
}

func (h *EventHandler) sendPlayerList(so io.Socket) {
	if players, err := h.service.Players(); err != nil {
		log.Println("player:request-remotes event error: " + err.Error())
	} else if err := so.Emit("remote-player:list", players); err != nil {
		log.Println("player:request-remotes event error: " + err.Error())
	}
}
