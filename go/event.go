package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"

	engineio "github.com/googollee/go-engine.io"
	io "github.com/googollee/go-socket.io"
)

// EventHandler handle socket.io events
type EventHandler struct {
	server   *io.Server
	service  *PlayerLocationService
	sessions *SessionManager
}

type Connections map[string]engineio.Conn

type SessionManager struct {
	connections atomic.Value
}

func NewSessionManager() *SessionManager {
	sessions := &SessionManager{}
	sessions.connections.Store(make(Connections))
	return sessions
}

func (sm *SessionManager) Get(id string) engineio.Conn {
	conns := sm.connections.Load().(Connections)
	return conns[id]
}

func (sm *SessionManager) Set(id string, conn engineio.Conn) {
	conns := sm.connections.Load().(Connections)
	conns[id] = conn
	sm.connections.Store(conns)
}

func (sm *SessionManager) Remove(id string) {
	conns := sm.connections.Load().(Connections)
	delete(conns, id)
	sm.connections.Store(conns)
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
		player := h.newPlayer(so, channel)

		h.sendPlayerList(so)

		so.On("player:request-list", func() {
			h.sendPlayerList(so)
		})

		so.On("player:update", func(msg string) {
			playerID := player.ID
			if err := json.Unmarshal([]byte(msg), player); err != nil {
				log.Panicln("player:update event error", err.Error())
				return
			}
			player.ID = playerID
			h.updatePlayer(so, player, channel)
		})

		so.On("admin:disconnect", func(playerID string) {
			log.Printf("admin:disconnect >%s< \n", playerID)

			conn := h.sessions.Get(playerID)
			if conn != nil {
				conn.Close()
			} else {
				player := &Player{ID: playerID}
				h.removePlayer(so, player, channel)
			}
		})

		so.On("disconnection", func() {
			h.removePlayer(so, player, channel)
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
	go log.Println("new player connected", player)
	return player
}

func (h *EventHandler) updatePlayer(so io.Socket, player *Player, channel string) {
	go log.Println("player:updated", player)
	so.Emit("player:updated", player)
	so.BroadcastTo(channel, "remote-player:updated", player)
	h.service.Update(player)
}

func (h *EventHandler) removePlayer(so io.Socket, player *Player, channel string) {
	if err := so.BroadcastTo(channel, "remote-player:destroy", player); err != nil {
		log.Panicln("Error broadcasting remote-player:destroy", channel, err)
	}
	if err := so.Emit("remote-player:destroy", player); err != nil {
		log.Println("Error broadcasting remote-player:destroy", channel, err)
	}
	h.service.Remove(player)
	go log.Println("--> diconnected", player)
}

func (h *EventHandler) sendPlayerList(so io.Socket) {
	if players, err := h.service.All(); err == nil {
		so.Emit("remote-player:list", players)
	} else {
		go log.Println("--> error to get players: ", err)
	}
}
