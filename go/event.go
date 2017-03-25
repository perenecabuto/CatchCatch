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
	return handler.bindEvents()
}

func (h *EventHandler) bindEvents() *EventHandler {
	h.server.On("connection", func(so io.Socket) {
		channel := "main"
		player, err := h.newPlayer(so, channel)
		if err != nil {
			log.Println("error to create player", err)
			h.sessions.Get(so.Id()).Close()
			return
		}
		log.Println("new player connected", player)

		go h.sendPlayerList(so)

		so.On("player:request-list", func() {
			h.sendPlayerList(so)
		})
		so.On("player:update", func(msg string) {
			playerID := player.ID
			if err := json.Unmarshal([]byte(msg), player); err != nil {
				log.Println("player:update event error: " + err.Error())
				return
			}
			player.ID = playerID
			h.updatePlayer(so, player, channel)
		})
		so.On("disconnection", func() {
			h.removePlayer(player, channel)
		})

		so.On("admin:clear", func(playerID string) {
			h.service.client.FlushDb()
			h.sessions.CloseAll()
		})
		so.On("admin:disconnect", func(playerID string) {
			log.Println("admin:disconnect", playerID)
			if conn := h.sessions.Get(playerID); conn != nil {
				conn.Close()
			} else {
				player := &Player{ID: playerID}
				h.removePlayer(player, channel)
			}
		})
	})

	return h
}

func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.server.ServeHTTP(w, r)
}

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

func (h *EventHandler) updatePlayer(so io.Socket, player *Player, channel string) {
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
