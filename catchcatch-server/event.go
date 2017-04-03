package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	io "github.com/googollee/go-socket.io"
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
	server.On("connection", handler.onConnection())
	return handler
}

func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.server.ServeHTTP(w, r)
}

// Event handlers

func (h *EventHandler) onConnection() func(so io.Socket) {
	return func(so io.Socket) {
		channel := "main"
		player, err := h.newPlayer(so, channel)
		if err != nil {
			log.Println("error to create player", err)
			h.sessions.Get(so.Id()).Close()
			return
		}
		log.Println("new player connected", player)
		go h.sendPlayerList(so)

		so.On("player:request-list", h.onPlayerRequestList(so))
		so.On("player:update", h.onPlayerUpdate(player, channel, so))
		so.On("disconnection", h.onPlayerDisconnect(player, channel))

		so.On("admin:disconnect", h.onDisconnectByID(channel))
		so.On("admin:feature:add", h.onAddFeature(channel))
		so.On("admin:feature:request-list", h.onRequestFeatures(so))
		so.On("admin:clear", h.onClear())
	}
}

func (h *EventHandler) onPlayerDisconnect(player *Player, channel string) func(string) {
	return func(string) {
		log.Println("player:disconnect", player.ID)
		if conn := h.sessions.Get(player.ID); conn != nil {
			conn.Close()
		}
		h.removePlayer(player, channel)
	}
}

func (h *EventHandler) onDisconnectByID(channel string) func(string) {
	return func(id string) {
		log.Println("admin:disconnect ", id)
		callback := h.onPlayerDisconnect(&Player{ID: id}, channel)
		callback("")
	}
}

func (h *EventHandler) onPlayerUpdate(player *Player, channel string, so io.Socket) func(string) {
	return func(msg string) {
		playerID := player.ID
		if err := json.Unmarshal([]byte(msg), player); err != nil {
			log.Println("player:update event error: " + err.Error())
			return
		}
		player.ID = playerID
		h.updatePlayer(player, channel, so)
	}
}

func (h *EventHandler) onPlayerRequestList(so io.Socket) func(string) {
	return func(string) {
		h.sendPlayerList(so)
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
		if err := h.addFeature(channel, group, name, geojson); err != nil {
			log.Println("Error to create geofence: ", err)
		}
	}
}

func (h *EventHandler) onRequestFeatures(so io.Socket) func(string) {
	return func(group string) {
		if err := h.sendFeatures(group, so); err != nil {
			log.Println("Error on sendFeatures:", err)
		}
	}
}

// Actions

func (h *EventHandler) newPlayer(so io.Socket, channel string) (player *Player, err error) {
	player = &Player{so.Id(), 0, 0}
	if err := h.service.Register(player); err != nil {
		return nil, errors.New("could not register: " + err.Error())
	}

	so.Join(channel)
	so.Emit("player:registred", player)
	so.BroadcastTo(channel, "remote-player:new", player)
	return player, nil
}

func (h *EventHandler) updatePlayer(player *Player, channel string, so io.Socket) {
	// go log.Println("player:updated", player)
	so.Emit("player:updated", player)
	so.BroadcastTo(channel, "remote-player:updated", player)
	h.service.Update(player)
}

func (h *EventHandler) removePlayer(player *Player, channel string) {
	h.server.BroadcastTo(channel, "remote-player:destroy", player)
	h.service.Remove(player)
	go log.Println("--> diconnected", player)
}

func (h *EventHandler) sendPlayerList(so io.Socket) error {
	players, err := h.service.Players()
	if err != nil {
		return err
	}
	return so.Emit("remote-player:list", players)
}

func (h *EventHandler) sendFeatures(group string, so io.Socket) error {
	features, err := h.service.Features(group)
	if err != nil {
		return err
	}
	return so.Emit("admin:feature:list", features)
}

func (h *EventHandler) addFeature(channel, group, name, geojson string) error {
	feature, err := h.service.AddFeature(group, name, geojson)
	if err != nil {
		return err
	}
	log.Println("Added feature", feature)
	h.server.BroadcastTo(channel, "admin:feature:added", feature)
	return nil
}
