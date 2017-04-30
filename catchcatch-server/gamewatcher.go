package main

import (
	"context"
	"errors"
	"log"
	"time"
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
			g.SetPlayer(p, gw.wss)
		case Inside:
			g.SetPlayer(p, gw.wss)
		case Exit:
			g.RemovePlayer(p, gw.wss)
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
			game := NewGame(gameID, gameDuration)
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
		if err := gw.wss.Emit(d.FeatID, "checkpoint:detected", d); err != nil {
			log.Println("Error to notify player", d.FeatID, err)
		}
		gw.wss.Broadcast("admin:feature:checkpoint", d)
	})
	if err != nil {
		log.Println("Error to stream geofence:event", err)
	}
}
