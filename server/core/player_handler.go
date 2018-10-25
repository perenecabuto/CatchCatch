package core

import (
	"context"
	"log"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/websocket"
)

const (
	EventPlayerRegistered = "player:registered"
	EventPlayerDisconnect = "player:disconnect"
	EventPlayerUpdate     = "player:update"
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
	// err = h.onGamesAround(ctx, wss)
	// if err != nil {
	// 	return err
	// }
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
	c.On(EventPlayerUpdate, h.onPlayerUpdate(player, c))
	c.OnDisconnected(h.onPlayerDisconnect(player))

	return nil
}

func (h *PlayerHandler) onPlayerDisconnect(player *model.Player) func() {
	return func() {
		log.Println(EventPlayerDisconnect, player.ID)
		h.players.Remove(player.ID)
	}
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
	}
}

func (h *PlayerHandler) newPlayer(c *websocket.WSConnectionHandler) (player *model.Player, err error) {
	player = &model.Player{ID: c.ID, Lat: 0, Lon: 0}
	if err := h.players.Set(player); err != nil {
		return nil, errors.Wrapf(err, "could not register player:%s", player.ID)
	}
	c.Emit(&protobuf.Player{EventName: proto.String(EventPlayerRegistered), Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
	return player, nil
}

// func (h *PlayerHandler) onGamesAround(ctx context.Context, wss *websocket.WSServer) error {
// return h.games.OnGameAround(ctx, func(p model.Player, g service.GameWithCoords) error {
// 	event := proto.String("game:around")
// 	err := wss.Emit(p.ID, &protobuf.Feature{EventName: event, Id: &g.ID, Group: proto.String("game"), Coords: &g.Coords})
// 	if err != nil {
// 		log.Println("Error to emit", *event)
// 	}
// 	return nil
// })
// }

func (h *PlayerHandler) onGameEvents(ctx context.Context, wss *websocket.WSServer) error {
	return h.games.OnGameEvent(ctx, func(p *GameEventPayload) error {
		switch p.Event {
		case GameStarted:
			wss.Emit(p.PlayerID, &protobuf.GameInfo{Id: &p.Game,
				EventName: proto.String(GameStarted.String()), Game: &p.Game,
				Role: proto.String(p.PlayerRole.String())})

		case GamePlayerNearToTarget:
			wss.Emit(p.PlayerID, &protobuf.Distance{Id: &p.Game,
				EventName: proto.String(GamePlayerNearToTarget.String()), Dist: &p.DistToTarget})

		case GamePlayerLose:
			wss.Emit(p.PlayerID, &protobuf.Simple{Id: &p.Game,
				EventName: proto.String(GamePlayerLose.String())})

		case GamePlayerWin:
			wss.Emit(p.PlayerID, &protobuf.Distance{Id: &p.Game,
				EventName: proto.String(GamePlayerWin.String()), Dist: &p.DistToTarget})

		case GameFinished:
			rank := p.Rank
			playersRank := make([]*protobuf.PlayerRank, len(rank.PlayerRank))
			for i, pr := range rank.PlayerRank {
				playersRank[i] = &protobuf.PlayerRank{Player: proto.String(pr.Player.ID), Points: proto.Int32(int32(pr.Points))}
			}
			wss.Emit(p.PlayerID, &protobuf.GameRank{
				EventName: proto.String(GameFinished.String()),
				Id:        &rank.Game, Game: &rank.Game,
				PlayersRank: playersRank,
			})
		}

		return nil
	})
}
