package core

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/websocket"
)

const (
	// MinPlayersPerGame ...
	MinPlayersPerGame = 3
	// DefaultGameDuration ...
	DefaultGameDuration = time.Minute
)

// GameWatcher is made to start/stop games by player presence
// and notify players events to each game by geo position
type GameWatcher struct {
	serverID string
	wss      *websocket.WSServer
	service  service.GameService
}

// NewGameWatcher builds GameWatecher
func NewGameWatcher(serverID string, service service.GameService, wss *websocket.WSServer) *GameWatcher {
	return &GameWatcher{serverID, wss, service}
}

// WatchGameEvents observers game events and notify players
// TODO: monitor game watches
func (gw *GameWatcher) WatchGameEvents(ctx context.Context) error {
	return gw.service.ObserveGamesEvents(ctx, func(g game.Game, evt game.Event) error {
		p := evt.Player
		switch evt.Name {
		case game.GameStarted:
			for _, p := range g.Players() {
				gw.wss.Emit(p.ID, &protobuf.GameInfo{
					EventName: proto.String("game:started"),
					Id:        &g.ID, Game: &g.ID,
					Role: proto.String(string(p.Role))})
			}

		case game.GamePlayerNearToTarget:
			gw.wss.Emit(p.ID, &protobuf.Distance{EventName: proto.String("game:target:near"), Dist: &p.DistToTarget})

		case game.GamePlayerLoose:
			gw.wss.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:loose"), Id: &g.ID})

		case game.GameTargetLoose:
			gw.wss.Emit(g.TargetID(), &protobuf.Simple{EventName: proto.String("game:loose"), Id: &g.ID})
			gw.wss.Emit(p.ID, &protobuf.Distance{EventName: proto.String("game:target:reached"),
				Dist: &p.DistToTarget})
			gw.sendGameRank(&g)

		case game.GameTargetWin:
			gw.wss.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:target:win")})
			gw.sendGameRank(&g)

		case game.GameFinished:
			gw.sendGameRank(&g)
		}

		return nil
	})
}

func (gw *GameWatcher) sendGameRank(g *game.Game) {
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
