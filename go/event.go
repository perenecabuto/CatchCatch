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
func NewEventHandler(server *io.Server, service *PlayerLocationService) *EventHandler {
	sessions := NewSessionManager()
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
		so.On("admin:clear", h.onClear())
	}
}

func (h *EventHandler) onPlayerDisconnect(player *Player, channel string) func() {
	return func() {
		if conn := h.sessions.Get(player.ID); conn != nil {
			conn.Close()
		}
		h.removePlayer(player, channel)
	}
}

func (h *EventHandler) onDisconnectByID(channel string) func(id string) {
	return func(id string) {
		log.Println("admin:disconnect ", id)
		callback := h.onPlayerDisconnect(&Player{ID: id}, channel)
		callback()
	}
}

func (h *EventHandler) onPlayerUpdate(player *Player, channel string, so io.Socket) func(msg string) {
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

func (h *EventHandler) onPlayerRequestList(so io.Socket) func() {
	return func() {
		h.sendPlayerList(so)
	}
}

func (h *EventHandler) onClear() func() {
	return func() {
		h.service.client.FlushDb()
		h.sessions.CloseAll()
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
	players, err := h.service.All()
	if err != nil {
		return err
	}
	return so.Emit("remote-player:list", players)
}
