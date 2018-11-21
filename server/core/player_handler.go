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

// PlayerHandler events
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
	err := h.listenToGameEvents(ctx, wss)
	if err != nil {
		return err
	}
	return nil
}

// OnConnection handles game and admin connection events
func (h *PlayerHandler) OnConnection(ctx context.Context, c *websocket.WSConnectionHandler) error {
	player, err := h.newPlayer(c)
	if err != nil {
		return errors.Wrap(err, "error to create player")
	}
	log.Println("[PlayerHandler] player connected", player)
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
		lat, lon := msg.GetLat(), msg.GetLon()
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
	c.Emit(&protobuf.Player{EventName: EventPlayerRegistered, Id: player.ID, Lon: player.Lon, Lat: player.Lat})
	return player, nil
}

func (h *PlayerHandler) listenToGameEvents(ctx context.Context, wss *websocket.WSServer) error {
	return h.games.OnGameEvent(ctx, func(p *GameEventPayload) error {
		switch p.Event {
		case GameStarted:
			wss.Emit(p.PlayerID, &protobuf.GameInfo{Id: p.Game,
				EventName: GameStarted.String(), Game: p.Game,
				Role: p.PlayerRole.String()})

		case GamePlayerNearToTarget:
			wss.Emit(p.PlayerID, &protobuf.Distance{Id: p.Game,
				EventName: GamePlayerNearToTarget.String(), Dist: p.DistToTarget})

		case GamePlayerLose:
			wss.Emit(p.PlayerID, &protobuf.Simple{Id: p.Game,
				EventName: GamePlayerLose.String()})

		case GamePlayerWin:
			wss.Emit(p.PlayerID, &protobuf.Distance{Id: p.Game,
				EventName: GamePlayerWin.String(), Dist: p.DistToTarget})

		case GameFinished:
			rank := p.Rank
			playersRank := make([]*protobuf.PlayerRank, len(rank.PlayerRank))
			for i, pr := range rank.PlayerRank {
				playersRank[i] = &protobuf.PlayerRank{Player: pr.Player.ID, Points: int32(pr.Points)}
			}
			wss.Emit(p.PlayerID, &protobuf.GameRank{
				EventName: GameFinished.String(),
				Id:        rank.Game, Game: rank.Game,
				PlayersRank: playersRank,
			})
		}

		return nil
	})
}
