package main

import (
	"context"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/protobuf"
)

const (
	// DefaultWatcherRange set the watcher radar radius size
	DefaultWatcherRange = 5000
	// MinPlayersPerGame ...
	MinPlayersPerGame = 3
	// DefaultGameDuration ...
	DefaultGameDuration = time.Minute
)

// GameWatcher is made to start/stop games by player presence
// and notify players events to each game by geo position
type GameWatcher struct {
	serverID string
	wss      *WSServer
	service  GameService
}

// NewGameWatcher builds GameWatecher
func NewGameWatcher(serverID string, service GameService, wss *WSServer) *GameWatcher {
	return &GameWatcher{serverID, wss, service}
}

// WatchGameEventsForever run WatchGameEvents
// if any error occur log it and run again, forever \o/
func (gw *GameWatcher) WatchGameEventsForever(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := gw.WatchGameEvents(ctx); err != nil {
				log.Panic("WatchGamesForever:error:", err)
			}
		}
	}
}

// WatchGameEvents observers game events and notify players
// TODO: monitor game watches
func (gw *GameWatcher) WatchGameEvents(ctx context.Context) error {
	return gw.service.ObserveGamesEvents(ctx, func(game *Game, evt *GameEvent) error {
		p := evt.Player
		switch evt.Name {
		case GameStarted:
			for _, p := range game.Players() {
				gw.wss.Emit(p.ID, &protobuf.GameInfo{
					EventName: proto.String("game:started"),
					Id:        &game.ID,
					Game:      &game.ID, Role: proto.String(string(p.Role))})

			}

		case GamePlayerNearToTarget:
			gw.wss.Emit(p.ID, &protobuf.Distance{EventName: proto.String("game:target:near"), Dist: &p.DistToTarget})

		case GamePlayerLoose:
			gw.wss.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:loose"), Id: &game.ID})

		case GameTargetLoose:
			gw.wss.Emit(game.targetID, &protobuf.Simple{EventName: proto.String("game:loose"), Id: &game.ID})
			gw.wss.Emit(p.ID, &protobuf.Distance{EventName: proto.String("game:target:reached"),
				Dist: &p.DistToTarget})
			gw.sendGameRank(game)

		case GameTargetWin:
			gw.wss.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:target:win")})
			gw.sendGameRank(game)

		case GameFinished:
			gw.sendGameRank(game)
		}

		return nil
	})
}

func (gw *GameWatcher) sendGameRank(g *Game) {
	rank := g.Rank()
	playersRank := make([]*protobuf.PlayerRank, len(rank.PlayerRank))
	for i, pr := range rank.PlayerRank {
		playersRank[i] = &protobuf.PlayerRank{Player: &pr.Player, Points: proto.Int32(int32(pr.Points))}
	}
	gw.wss.EmitTo(rank.PlayerIDs, &protobuf.GameRank{
		EventName: proto.String("game:finish"),
		Id:        &rank.Game,
		Game:      &rank.Game, PlayersRank: playersRank,
	})
}
