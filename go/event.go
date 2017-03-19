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

// NewEventHandler EventHandler builder
func NewEventHandler(server *io.Server, service *PlayerLocationService) *EventHandler {
	handler := &EventHandler{server, service}
	return handler.bindEvents()
}

func (h *EventHandler) bindEvents() *EventHandler {
	h.server.On("connection", func(so io.Socket) {
		channel := "main"
		player := h.newPlayer(so, channel)

		h.sendPlayerList(so)

		so.On("player:update", func(msg string) {
			if err := json.Unmarshal([]byte(msg), player); err != nil {
				log.Println("player:update event error", err.Error())
				return
			}
			log.Println("player:updated", player)
			so.Emit("player:updated", player)
			so.BroadcastTo(channel, "remote-player:updated", player)
			h.service.Update(player)
		})

		so.On("disconnection", func() {
			if err := so.Leave(channel); err != nil {
				log.Fatal("Error leaving channel", channel, err)
			}
			if err := so.BroadcastTo(channel, "remote-player:destroy", player); err != nil {
				log.Fatal("Error broadcasting remote-player:destroy", channel, err)
			}
			h.service.Remove(player)
			log.Println("----> diconnected", player)
			player = nil
		})
	})

	return h
}

func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.server.ServeHTTP(w, r)
}

func (h *EventHandler) newPlayer(so io.Socket, channel string) *Player {
	player := &Player{so.Id(), 0, 0}
	so.Join(channel)
	if err := h.service.Register(player); err != nil {
		log.Fatal("could not register:", err)
	}
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
