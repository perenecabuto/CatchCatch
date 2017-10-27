package main

import (
	"context"
	"log"
	"runtime/debug"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/protobuf"
)

const (
	// MinPlayersPerGame ...
	MinPlayersPerGame = 3
	// DefaultGameDuration ...
	DefaultGameDuration = time.Minute
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
	wss    *WSServer
	stream EventStream
	Clear  context.CancelFunc
}

// NewGameWatcher builds GameWatecher
func NewGameWatcher(stream EventStream, wss *WSServer) *GameWatcher {
	return &GameWatcher{make(map[string]*GameContext), wss, stream, func() {}}
}

// observeGamePlayers events
// TODO: monitor game player watches
func (gw *GameWatcher) observeGamePlayers(ctx context.Context, g *Game) error {
	return gw.stream.StreamIntersects(ctx, "player", "geofences", g.ID, func(d *Detection) error {
		switch d.Intersects {
		case Enter:
			if err := g.SetPlayer(d.FeatID, d.Lat, d.Lon); err != nil {
				return err
			}
		case Inside:
			if err := g.SetPlayer(d.FeatID, d.Lat, d.Lon); err != nil {
				return err
			}
		case Exit:
			g.RemovePlayer(d.FeatID)
		}
		return nil
	})
}

// WatchGames starts this gamewatcher to listen to player events over games
// TODO: destroy the game after it finishes
// TODO: monitor game watches
func (gw *GameWatcher) WatchGames(ctx context.Context) error {
	var watcherCtx context.Context
	watcherCtx, gw.Clear = context.WithCancel(ctx)
	defer gw.Clear()

	return gw.stream.StreamNearByEvents(watcherCtx, "player", "geofences", 0, func(d *Detection) error {
		gameID := d.NearByFeatID
		if gameID == "" {
			return nil
		}

		go func() {
			if err := gw.watchGame(watcherCtx, gameID); err != nil {
				log.Println(err)
				gw.games[gameID].cancel()
			}
		}()
		return nil
	})
}

// WatchGamesForever restart game wachter util context done
func (gw *GameWatcher) WatchGamesForever(ctx context.Context) error {
	done := ctx.Done()
	for {
		select {
		case <-done:
			return nil
		default:
			if err := gw.WatchGames(ctx); err != nil {
				return err
			}
		}
	}
}

func (gw *GameWatcher) watchGame(ctx context.Context, gameID string) error {
	_, exists := gw.games[gameID]
	if exists {
		return nil
	}
	g := NewGame(gameID, DefaultGameDuration, gw)
	gCtx, cancel := context.WithCancel(ctx)
	gw.games[gameID] = &GameContext{game: g, cancel: func() {
		delete(gw.games, gameID)
		cancel()
	}}

	errChan := make(chan error)
	go func() {
		errChan <- gw.observeGamePlayers(gCtx, g)
	}()
	go func() {
		if err := gw.startGameWhenReady(gCtx, g); err != nil {
			errChan <- err
		}
	}()
	if err := <-errChan; err != nil {
		gw.games[gameID].cancel()
		return err
	}
	return nil
}

func (gw *GameWatcher) startGameWhenReady(ctx context.Context, g *Game) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			ready := len(g.players) >= MinPlayersPerGame
			if ready {
				return g.Start(ctx)
			}
		}
	}
}

// WatchCheckpoints ...
func (gw *GameWatcher) WatchCheckpoints(ctx context.Context) {
	err := gw.stream.StreamNearByEvents(ctx, "player", "checkpoint", 1000, func(d *Detection) error {
		if d.Intersects == Exit {
			return nil
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
		return gw.wss.Broadcast(payload)
	})
	if err != nil {
		log.Println("Error to stream geofence:event", err)
		debug.PrintStack()
	}
}

// game callbacks

// OnGameStarted implements GameEvent.OnGameStarted
func (gw *GameWatcher) OnGameStarted(g *Game, p GamePlayer) {
	gw.wss.Emit(p.ID, &protobuf.GameInfo{
		EventName: proto.String("game:started"),
		Id:        &g.ID,
		Game:      &g.ID, Role: proto.String(string(p.Role))})
}

// OnTargetWin implements GameEvent.OnTargetWin
func (gw *GameWatcher) OnTargetWin(p GamePlayer) {
	gw.wss.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:target:win")})
}

// OnGameFinish implements GameEvent.OnGameFinish
func (gw *GameWatcher) OnGameFinish(rank GameRank) {
	log.Printf("gamewatcher:stop:game:%s", rank.Game)
	gw.games[rank.Game].cancel()

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
func (gw *GameWatcher) OnPlayerLoose(g *Game, p GamePlayer) {
	gw.wss.Emit(p.ID, &protobuf.Simple{EventName: proto.String("game:loose"), Id: &g.ID})
}

// OnTargetReached implements GameEvent.OnTargetReached
func (gw *GameWatcher) OnTargetReached(p GamePlayer, dist float64) {
	gw.wss.Emit(p.ID, &protobuf.Distance{EventName: proto.String("game:target:reached"),
		Dist: &dist})
}

// OnPlayerNearToTarget implements GameEvent.OnPlayerNearToTarget
func (gw *GameWatcher) OnPlayerNearToTarget(p GamePlayer, dist float64) {
	gw.wss.Emit(p.ID, &protobuf.Distance{EventName: proto.String("game:target:near"),
		Dist: &dist})
}
