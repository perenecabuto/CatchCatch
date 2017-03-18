package main

import (
	"encoding/json"
	"log"
	"net/http"

	io "github.com/googollee/go-socket.io"
)

// EventHandler handle socket.io events
type EventHandler struct {
	server  *io.Server
	service *PlayerLocationService
}

func NewEventHandler(server *io.Server, service *PlayerLocationService) *EventHandler {
	return &EventHandler{server, service}
}

func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.server.On("connection", func(so io.Socket) {
		channel := "main"
		player := h.newPlayer(so, channel)

		h.sendPlayerList(so)

		so.On("player:update", func(msg string) {
			println(msg)
			if err := json.Unmarshal([]byte(msg), player); err != nil {
				log.Println("player:update event error", err.Error())
				return
			}
			so.Emit("player:updated", player)
			so.BroadcastTo(channel, "remote-player:updated", player)
			h.service.Update(player)
		})

		so.On("disconnection", func() {
			so.Leave(channel)
			so.BroadcastTo(channel, "remote-player:destroy", player)
			h.service.Remove(player)
			log.Println("diconnected", player)
		})
	})

	h.server.ServeHTTP(w, r)
}

func (h *EventHandler) newPlayer(so io.Socket, channel string) *Player {
	player := &Player{so.Id(), 0, 0}
	so.Join(channel)
	h.service.Register(player)
	so.Emit("player:registred", player)
	so.BroadcastTo(channel, "remote-player:new", player)
	log.Println("new player connected", player)
	return player
}

func (h *EventHandler) sendPlayerList(so io.Socket) {
	if players, err := h.service.All(); err == nil {
		so.Emit("remote-player:list", players)
	} else {
		log.Println("--> error to get players: ", err)
	}
}
