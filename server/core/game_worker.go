package core

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service"
)

var (
	// ErrGameStoped happens when game can't change anymore
	ErrGameStoped = errors.New("game stoped")
)

// GameWorker observe manage and run games
type GameWorker struct {
	serverID string
	service  service.GameService
}

// NewGameWorker creates GameWorker
func NewGameWorker(serverID string, service service.GameService) *GameWorker {
	return &GameWorker{serverID, service}
}

// WatchGames starts this Worker to listen to player events over games
// TODO: monitor game watches
func (gw GameWorker) WatchGames(ctx context.Context) error {
	return gw.service.ObservePlayersCrossGeofences(ctx, func(gameID string, _ model.Player) error {
		go func() {
			if err := gw.watchGame(ctx, gameID); err != nil {
				log.Println("Worker:WatchGames:error:", err)
			}
		}()
		return nil
	})
}

// TODO: monitor game player watches
// TODO: before starting idle game watcher verify if the server watcher is the same of this server
// TODO: start watcher if the server is the watcher and isn't running
func (gw GameWorker) watchGame(ctx context.Context, gameID string) error {
	if running, err := gw.service.IsGameRunning(gameID); err != nil {
		return err
	} else if running {
		return nil
	}
	log.Printf("GameWatcher:create:%s", gameID)

	if err := gw.service.Create(gameID, gw.serverID); err != nil {
		return err
	}

	g := game.NewGame(gameID)
	gCtx, stop := context.WithCancel(ctx)

	evtChan := make(chan game.GameEvent, 100)
	defer close(evtChan)
	gameTimer := time.NewTimer(time.Hour)
	defer gameTimer.Stop()
	gameHealthCheckTicker := time.NewTicker(30 * time.Second)
	defer gameHealthCheckTicker.Stop()

	go func() {
		err := gw.service.ObserveGamePlayers(gCtx, g.ID, func(p model.Player, exit bool) error {
			var evt game.GameEvent
			var err error
			if exit {
				evt, err = g.RemovePlayer(p.ID)
			} else {
				evt, err = g.SetPlayer(p.ID, p.Lat, p.Lon)
			}
			if err != nil {
				return err
			}
			if evt.Name != game.GameNothingHappens {
				evtChan <- evt
			}
			return nil
		})
		if err != nil {
			log.Println("Worker:watchGame:error:", err)
			stop()
		}
	}()

	for {
		select {
		case evt := <-evtChan:
			log.Printf("GameWorker:%s:gameevent:%-v", gameID, evt)

			switch evt.Name {
			case game.GameTargetWin, game.GameTargetLoose, game.GameLastPlayerDetected, game.GameRunningWithoutPlayers:
				gw.service.Update(g, gw.serverID, evt)
				stop()
			case game.GamePlayerNearToTarget:
				gw.service.Update(g, gw.serverID, evt)
			case game.GamePlayerAdded, game.GamePlayerRemoved:
				// TODO: monitor game start
				ready := !g.Started() && len(g.Players()) >= MinPlayersPerGame
				if ready {
					gameTimer = time.NewTimer(5 * time.Minute)
					evt = g.Start()
					gw.service.Update(g, gw.serverID, evt)
				}
			}
		case <-gameHealthCheckTicker.C:
			// TODO do not send events for this
			err := gw.service.Update(g, gw.serverID, game.GameEventNothing)
			if err != nil {
				log.Println("Worker:watchGame:healthcheck:error:", err)
			}
		case <-gameTimer.C:
			stop()
		case <-gCtx.Done():
			log.Printf("gamewatcher:stop:game:%s", gameID)
			stop()
			evt := g.Stop()
			gw.service.Update(g, gw.serverID, evt)
			gw.service.Remove(gameID)
			return nil
		}
	}
}
