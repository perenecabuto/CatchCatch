package core

import (
	"context"
	"errors"
	"log"

	"github.com/golang/protobuf/proto"

	"github.com/perenecabuto/CatchCatch/server/game"
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

func (h *PlayerHandler) OnStart(ctx context.Context, wss *websocket.WSServer) error {
	return nil
}

// OnConnection handles game and admin connection events
func (h *PlayerHandler) OnConnection(ctx context.Context, c *websocket.WSConnectionHandler) error {
	player, err := h.newPlayer(c)
	if err != nil {
		log.Println("error to create player", err)
		return err
	}

	log.Println("new player connected", player)
	c.On("player:request-games", h.onPlayerRequestGames(player, c))
	c.On("player:update", h.onPlayerUpdate(player, c))
	c.OnDisconnected(func() {
		h.onPlayerDisconnect(player)
	})

	return nil
}

func (h *PlayerHandler) onPlayerDisconnect(player *model.Player) {
	log.Println("player:disconnect", player.ID)
	h.players.Remove(player.ID)
}

func (h *PlayerHandler) onPlayerUpdate(player *model.Player, c *websocket.WSConnectionHandler) func([]byte) {
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

func (h *PlayerHandler) onPlayerRequestGames(player *model.Player, c *websocket.WSConnectionHandler) func([]byte) {
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

func (h *PlayerHandler) newPlayer(c *websocket.WSConnectionHandler) (player *model.Player, err error) {
	player = &model.Player{ID: c.ID, Lat: 0, Lon: 0}
	if err := h.players.Set(player); err != nil {
		return nil, errors.New("could not register: " + err.Error())
	}
	c.Emit(&protobuf.Player{EventName: proto.String("player:registered"), Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
	return player, nil
}

// WatchGameEvents notify player about game events
func (h *PlayerHandler) WatchGameEvents(ctx context.Context) error {
	return h.games.ObserveGamesEvents(ctx, func(g *game.Game, evt game.Event) error {
		p := evt.Player
		switch evt.Name {
		case game.GameStarted:
			for _, p := range g.Players() {
				h.server.Emit(p.ID, &protobuf.GameInfo{
					EventName: proto.String("game:started"),
					Id:        &g.ID, Game: &g.ID,
					Role: proto.String(string(p.Role))})
			}

		case game.GamePlayerNearToTarget:
			h.server.Emit(p.ID, &protobuf.Distance{EventName: proto.String("game:target:near"), Dist: &p.DistToTarget})

		case game.GamePlayerLoose:
			h.server.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:loose"), Id: &g.ID})

		case game.GameTargetLoose:
			h.server.Emit(g.TargetID(), &protobuf.Simple{EventName: proto.String("game:loose"), Id: &g.ID})
			h.server.Emit(p.ID, &protobuf.Distance{EventName: proto.String("game:target:reached"),
				Dist: &p.DistToTarget})
			h.sendGameRank(g)

		case game.GameTargetWin:
			h.server.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:target:win")})
			h.sendGameRank(g)

		case game.GameFinished:
			h.sendGameRank(g)
		}

		return nil
	})
}

func (h *PlayerHandler) sendGameRank(g *game.Game) {
	rank := g.Rank()
	playersRank := make([]*protobuf.PlayerRank, len(rank.PlayerRank))
	for i, pr := range rank.PlayerRank {
		playersRank[i] = &protobuf.PlayerRank{Player: &pr.Player, Points: proto.Int32(int32(pr.Points))}
	}
	h.server.EmitTo(rank.PlayerIDs, &protobuf.GameRank{
		EventName: proto.String("game:finish"),
		Id:        &rank.Game,
		Game:      &rank.Game, PlayersRank: playersRank,
	})
}
