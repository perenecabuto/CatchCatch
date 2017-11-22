package main

import (
	"errors"
	"log"

	"github.com/golang/protobuf/proto"

	"github.com/perenecabuto/CatchCatch/catchcatch-server/model"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/protobuf"
)

//TODO: separate player events and admin events
//TODO: separate player and admin routes
//TODO: set game status on db

// EventHandler handle websocket events
type EventHandler struct {
	server  *WSServer
	service PlayerLocationService
}

// NewEventHandler EventHandler builder
func NewEventHandler(server *WSServer, service PlayerLocationService) *EventHandler {
	handler := &EventHandler{server, service}
	server.OnConnected(handler.onConnection)
	return handler
}

// Event handlers

func (h *EventHandler) onConnection(c *WSConnListener) {
	player, err := h.newPlayer(c)
	if err != nil {
		log.Println("error to create player", err)
		c.Close()
		return
	}

	log.Println("new player connected", player)

	c.On("player:request-games", h.onPlayerRequestGames(player, c))
	c.On("player:update", h.onPlayerUpdate(player, c))
	c.OnDisconnected(func() {
		h.onPlayerDisconnect(player)
	})

	c.On("admin:disconnect", h.onDisconnectByID(c))
	c.On("admin:feature:add", h.onAddFeature())
	c.On("admin:feature:request-remotes", h.onPlayerRequestRemotes(c))
	c.On("admin:feature:request-list", h.onRequestFeatures(c))
	c.On("admin:clear", h.onClear())
}

// Player events

func (h *EventHandler) onPlayerDisconnect(player *model.Player) {
	log.Println("player:disconnect", player.ID)
	h.service.Remove(player)
}

func (h *EventHandler) onPlayerUpdate(player *model.Player, c *WSConnListener) func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Player{}
		proto.Unmarshal(buf, msg)
		lat, lon := float64(float32(msg.GetLat())), float64(float32(msg.GetLon()))
		if lat == 0 || lon == 0 {
			return
		}
		player.Lat, player.Lon = lat, lon
		h.service.Update(player)

		c.Emit(&protobuf.Player{EventName: proto.String("player:updated"),
			Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
	}
}

func (h *EventHandler) onPlayerRequestRemotes(so *WSConnListener) func([]byte) {
	return func([]byte) {
		players, err := h.service.Players()
		if err != nil {
			log.Println("player:request-remotes event error: " + err.Error())
		}
		event := "remote-player:new"
		for _, p := range players {
			if p == nil {
				continue
			}
			err := so.Emit(&protobuf.Player{EventName: &event, Id: &p.ID, Lon: &p.Lon, Lat: &p.Lat})
			if err != nil {
				log.Println("player:request-remotes event error: " + err.Error())
			}
		}
	}
}

func (h *EventHandler) onPlayerRequestGames(player *model.Player, c *WSConnListener) func([]byte) {
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

func (h *EventHandler) onDisconnectByID(c *WSConnListener) func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Simple{}
		proto.Unmarshal(buf, msg)
		log.Println("admin:disconnect", msg.GetId())
		player := &model.Player{ID: msg.GetId()}
		err := h.service.Remove(player)
		if err == ErrFeatureNotFound {
			// Notify remote-player removal to ghost players on admin
			log.Println("admin:disconnect:force", msg.GetId())
			c.Emit(&protobuf.Player{EventName: proto.String("remote-player:destroy"),
				Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
		}
		h.server.Remove(msg.GetId())
	}
}

func (h *EventHandler) onClear() func([]byte) {
	return func([]byte) {
		// TODO: send this message by broaker
		// h.games.Clear()
		h.service.Clear()
		h.server.CloseAll()
	}
}

// Map events

func (h *EventHandler) onAddFeature() func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Feature{}
		proto.Unmarshal(buf, msg)

		_, err := h.service.SetFeature(msg.GetGroup(), msg.GetId(), msg.GetCoords())
		if err != nil {
			log.Println("Error to create feature:", err)
			return
		}
	}
}

func (h *EventHandler) onRequestFeatures(c *WSConnListener) func([]byte) {
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

func (h *EventHandler) newPlayer(c *WSConnListener) (player *model.Player, err error) {
	player = &model.Player{ID: c.ID, Lat: 0, Lon: 0}
	if err := h.service.Register(player); err != nil {
		return nil, errors.New("could not register: " + err.Error())
	}
	c.Emit(&protobuf.Player{EventName: proto.String("player:registered"), Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
	return player, nil
}
