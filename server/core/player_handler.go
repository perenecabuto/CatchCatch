package core

import (
	"errors"
	"log"

	"github.com/golang/protobuf/proto"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/websocket"
)

//TODO: set game status on db

// PlayerHandler handle websocket events
type PlayerHandler struct {
	server  *websocket.WSServer
	players service.PlayerLocationService
	games   service.GameService
}

// NewPlayerHandler PlayerHandler builder
func NewPlayerHandler(s *websocket.WSServer,
	p service.PlayerLocationService, g service.GameService) *PlayerHandler {
	handler := &PlayerHandler{s, p, g}
	return handler
}

// Event handlers

// OnConnection handles game and admin connection events
func (h *PlayerHandler) OnConnection(c *websocket.WSConnListener) {
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
}

func (h *PlayerHandler) onPlayerDisconnect(player *model.Player) {
	log.Println("player:disconnect", player.ID)
	h.players.Remove(player)
}

func (h *PlayerHandler) onPlayerUpdate(player *model.Player, c *websocket.WSConnListener) func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Player{}
		proto.Unmarshal(buf, msg)
		lat, lon := float64(float32(msg.GetLat())), float64(float32(msg.GetLon()))
		if lat == 0 || lon == 0 {
			return
		}
		player.Lat, player.Lon = lat, lon
		h.players.Set(player)

		c.Emit(&protobuf.Player{EventName: proto.String("player:updated"),
			Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
	}
}

func (h *PlayerHandler) onPlayerRequestGames(player *model.Player, c *websocket.WSConnListener) func([]byte) {
	return func([]byte) {
		go func() {
			games, err := h.games.GamesAround(*player)
			if err != nil {
				log.Println("Error to request games:", err)
				return
			}
			event := proto.String("game:around")
			for _, g := range games {
				err := c.Emit(&protobuf.Feature{EventName: event, Id: &g.ID, Group: proto.String("game"), Coords: &g.Coords})
				if err != nil {
					log.Println("Error to emit", *event, player)
				}
			}
		}()
	}
}

func (h *PlayerHandler) newPlayer(c *websocket.WSConnListener) (player *model.Player, err error) {
	player = &model.Player{ID: c.ID, Lat: 0, Lon: 0}
	if err := h.players.Set(player); err != nil {
		return nil, errors.New("could not register: " + err.Error())
	}
	c.Emit(&protobuf.Player{EventName: proto.String("player:registered"), Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
	return player, nil
}
