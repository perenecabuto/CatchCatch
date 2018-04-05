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

// PlayerHandler handle websocket events
type PlayerHandler struct {
	players service.PlayerLocationService
	games   *GameWorker
}

// NewPlayerHandler PlayerHandler builder
func NewPlayerHandler(p service.PlayerLocationService, g *GameWorker) *PlayerHandler {
	handler := &PlayerHandler{p, g}
	return handler
}

// OnStart add listeners for game events, games around players
func (h *PlayerHandler) OnStart(ctx context.Context, wss *websocket.WSServer) error {
	err := h.onGameEvents(ctx, wss)
	if err != nil {
		return err
	}
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

func (h *PlayerHandler) newPlayer(c *websocket.WSConnectionHandler) (player *model.Player, err error) {
	player = &model.Player{ID: c.ID, Lat: 0, Lon: 0}
	if err := h.players.Set(player); err != nil {
		return nil, errors.New("could not register: " + err.Error())
	}
	c.Emit(&protobuf.Player{EventName: proto.String("player:registered"), Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
	return player, nil
}

func (h *PlayerHandler) onGameEvents(ctx context.Context, wss *websocket.WSServer) error {
	return h.games.OnGameEvent(ctx, func(g *game.Game, evt game.Event) error {
		p := evt.Player

		switch evt.Name {
		case game.GameCreated:
		case game.GamePlayerAdded:
		case game.GamePlayerRemoved:
		case game.GameStarted:
			for _, p := range g.Players() {
				wss.Emit(p.ID, &protobuf.GameInfo{
					EventName: proto.String("game:started"),
					Id:        &g.ID, Game: &g.ID,
					Role: proto.String(string(p.Role))})
			}

		case game.GamePlayerNearToTarget:
			wss.Emit(p.ID, &protobuf.Distance{EventName: proto.String("game:target:near"), Dist: &p.DistToTarget})

		case game.GamePlayerLoose:
			wss.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:loose"), Id: &g.ID})

		case game.GameTargetLoose:
			wss.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:loose"), Id: &g.ID})

		case game.GameTargetWin:
			wss.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:target:win")})

		case game.GameRunningWithoutPlayers:

		case game.GameLastPlayerDetected:

		case game.GameFinished:
			rank := g.Rank()
			playersRank := make([]*protobuf.PlayerRank, len(rank.PlayerRank))
			for i, pr := range rank.PlayerRank {
				playersRank[i] = &protobuf.PlayerRank{Player: &pr.Player, Points: proto.Int32(int32(pr.Points))}
			}
			wss.Emit(p.ID, &protobuf.GameRank{
				EventName: proto.String("game:finish"),
				Id:        &rank.Game,
				Game:      &rank.Game, PlayersRank: playersRank,
			})
		}

		return nil
	})
}
