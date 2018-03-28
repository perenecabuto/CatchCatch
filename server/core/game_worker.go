package core

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/perenecabuto/CatchCatch/server/worker"

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
func NewGameWorker(serverID string, service service.GameService) worker.Worker {
	return &GameWorker{serverID, service}
}

func (gw GameWorker) ID() string {
	return "GameWorker"
}

// WatchGames starts this Worker to listen to player events over games
// TODO: monitor game watches
// TODO: monitor game player watches
// TODO: before starting idle game watcher verify if the server watcher is the same of this server
// TODO: start watcher if the server is the watcher and isn't running
func (gw GameWorker) Run(ctx context.Context, _ worker.TaskParams) error {
	return gw.service.ObservePlayersCrossGeofences(ctx, func(gameID string, _ model.Player) error {
		if running, err := gw.service.IsGameRunning(gameID); err != nil {
			return err
		} else if running {
			return nil
		}
		go func() {
			err := gw.watchGamePlayers(ctx, gameID)
			if err != nil {
				log.Println("Worker:WatchGames:error:", err)
			}
		}()
		return nil
	})
}

func (gw GameWorker) watchGamePlayers(ctx context.Context, gameID string) error {
	log.Printf("GameWatcher:create:%s", gameID)
	g, err := gw.service.Create(gameID, gw.serverID)
	if err != nil {
		return err
	}

	gCtx, stop := context.WithCancel(ctx)
	defer stop()
	gameTimer := time.NewTimer(time.Hour)
	defer gameTimer.Stop()
	gameHealthCheckTicker := time.NewTicker(30 * time.Second)
	defer gameHealthCheckTicker.Stop()
	evtChan := make(chan game.Event, 1)
	defer close(evtChan)

	go func() {
		err := gw.service.ObserveGamePlayers(gCtx, g.ID, func(p model.Player, exit bool) error {
			var evt game.Event
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
			log.Println("Error on ObserveGamePlayers", err)
			stop()
		}
	}()

	for {
		select {
		case evt, ok := <-evtChan:
			if !ok {
				stop()
				continue
			}
			log.Printf("GameWorker:%s:gameevent:%-v", g.ID, evt)

			switch evt.Name {
			case game.GameTargetWin,
				game.GameTargetLoose,
				game.GameLastPlayerDetected,
				game.GameRunningWithoutPlayers:

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
			err := gw.service.Update(g, gw.serverID, game.GameEventNothing)
			if err != nil {
				log.Println("GameWorker:watchGame:healthcheck:error:", err)
			}
		case <-gameTimer.C:
			log.Printf("GameWorker:watchGame:stop:game:%s", g.ID)
			stop()
		case <-gCtx.Done():
			log.Printf("GameWorker:watchGame:done:game:%s", g.ID)
			evt := g.Stop()
			// TODO store serverID in another store,
			// maybe a env kv store like etcd
			gw.service.Update(g, gw.serverID, evt)
			gw.service.Remove(g.ID)
			return nil
		}
	}
}
