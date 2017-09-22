package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/protobuf"
)

// GameContext stores game and its canel (and stop eventualy) function
type GameContext struct {
	game   *Game
	cancel context.CancelFunc
}

// GameWatcher is made to start/stop games by player presence
// and notify players events to each game by geo position
type GameWatcher struct {
	games  map[string]*GameContext
	wss    *WebSocketServer
	stream EventStream
}

// NewGameWatcher builds GameWatecher
func NewGameWatcher(stream EventStream, wss *WebSocketServer) *GameWatcher {
	return &GameWatcher{make(map[string]*GameContext), wss, stream}
}

// WatchGamePlayers events
func (gw *GameWatcher) WatchGamePlayers(ctx context.Context, g *Game) error {
	err := gw.stream.StreamIntersects(ctx, "player", "geofences", g.ID, func(d *Detection) {
		p := &Player{ID: d.FeatID, Lat: d.Lat, Lon: d.Lon}
		switch d.Intersects {
		case Enter:
			g.SetPlayer(p)
		case Inside:
			g.SetPlayer(p)
		case Exit:
			g.RemovePlayer(p)
		}
	})
	return err
}

// WatchGames starts this gamewatcher to listen to player events over games
func (gw *GameWatcher) WatchGames(ctx context.Context) error {
	defer gw.Clear()
	err := gw.stream.StreamNearByEvents(ctx, "player", "geofences", 0, func(d *Detection) {
		gameID := d.NearByFeatID
		if gameID == "" {
			return
		}
		_, exists := gw.games[gameID]
		if !exists {
			gameDuration := time.Minute
			game := NewGame(gameID, gameDuration, gw)
			gctx, cancel := context.WithCancel(ctx)
			log.Println("gamewatcher:create:game:", gameID, d.FeatID)
			gw.games[gameID] = &GameContext{game, cancel}

			go func() {
				if err := gw.WatchGamePlayers(gctx, game); err != nil {
					log.Printf("Error to start gamewatcher:%s - err: %v", game.ID, err)
				}

				log.Println("gamewatcher:destroy:game:", game.ID)
				gw.StopGame(game.ID)
			}()
		}
	})
	if err != nil {
		return errors.New("Error to stream geofence:event " + err.Error())
	}
	return nil
}

// Clear stop all started games
func (gw *GameWatcher) Clear() {
	log.Printf("gamewatcher:clear:games")
	for id := range gw.games {
		gw.StopGame(id)
	}
}

// StopGame stops a game and its watcher
func (gw *GameWatcher) StopGame(gameID string) {
	if _, exists := gw.games[gameID]; !exists {
		return
	}
	log.Printf("gamewatcher:stop:game:%s", gameID)
	gw.games[gameID].cancel()
	gw.games[gameID].game.Stop()
	delete(gw.games, gameID)
}

// WatchCheckpoints ...
func (gw *GameWatcher) WatchCheckpoints(ctx context.Context) {
	err := gw.stream.StreamNearByEvents(ctx, "player", "checkpoint", 1000, func(d *Detection) {
		if d.Intersects == Exit {
			return
		}
		payload := &protobuf.Detection{
			EventName:    proto.String("checkpoint:detected"),
			Id:           &d.FeatID,
			FeatId:       &d.FeatID,
			Lon:          &d.Lon,
			Lat:          &d.Lat,
			NearByFeatId: &d.NearByFeatID,
			NearByMeters: &d.NearByMeters,
		}

		if err := gw.wss.Emit(d.FeatID, payload); err != nil {
			log.Println("Error to notify player", d.FeatID, err)
		}
		payload.EventName = proto.String("admin:feature:checkpoint")
		gw.wss.Broadcast(payload)
	})
	if err != nil {
		log.Println("Error to stream geofence:event", err)
	}
}

// game callbacks

// OnGameStarted implements GameEvent.OnGameStarted
func (gw *GameWatcher) OnGameStarted(g *Game, p *Player, role string) {
	gw.wss.Emit(p.ID, &protobuf.GameInfo{
		EventName: proto.String("game:started"),
		Id:        &g.ID,
		Game:      &g.ID, Role: &role})
}

// OnTargetWin implements GameEvent.OnTargetWin
func (gw *GameWatcher) OnTargetWin(p *Player) {
	gw.wss.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:target:win")})
}

// OnGameFinish implements GameEvent.OnGameFinish
func (gw *GameWatcher) OnGameFinish(rank GameRank) {
	playersRank := make([]*protobuf.PlayerRank, len(rank.PlayerRank))
	for i, pr := range rank.PlayerRank {
		playersRank[i] = &protobuf.PlayerRank{Player: &pr.Player, Points: proto.Int32(int32(pr.Points))}
	}
	gw.wss.BroadcastTo(rank.PlayerIDs, &protobuf.GameRank{
		EventName: proto.String("game:finish"),
		Id:        &rank.Game,
		Game:      &rank.Game, PlayersRank: playersRank,
	})
}

// OnPlayerLoose implements GameEvent.OnPlayerLoose
func (gw *GameWatcher) OnPlayerLoose(g *Game, p *Player) {
	gw.wss.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:loose"), Id: &g.ID})
}

// OnTargetReached implements GameEvent.OnTargetReached
func (gw *GameWatcher) OnTargetReached(p *Player, dist float64) {
	gw.wss.Emit(p.ID, &protobuf.Distance{EventName: proto.String("game:target:reached"),
		Dist: &dist})
}

// OnPlayerNearToTarget implements GameEvent.OnPlayerNearToTarget
func (gw *GameWatcher) OnPlayerNearToTarget(p *Player, dist float64) {
	gw.wss.Emit(p.ID, &protobuf.Distance{EventName: proto.String("game:target:near"),
		Dist: &dist})
}
