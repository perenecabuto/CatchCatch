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

	MinPlayersPerGame = 3
)

// GameWorker observe manage and run games
type GameWorker struct {
	service service.GameService
}

// NewGameWorker creates GameWorker
func NewGameWorker(service service.GameService) worker.Worker {
	return &GameWorker{service}
}

func (gw GameWorker) ID() string {
	return "GameWorker"
}

// Run starts this Worker to listen to player events over games
// TODO: monitor game watches
// TODO: monitor game player watches
func (gw GameWorker) Run(ctx context.Context, params worker.TaskParams) error {
	gameID, ok := params["gameID"].(string)
	if !ok {
		return errors.New("gameID can't be empty")
	}
	coordinates, ok := params["coordinates"].(string)
	if !ok {
		return errors.New("game coordinates can't be empty")
	}
	// gw.service.Remove(gameID)

	log.Printf("GameWatcher:create:%s", gameID)
	g, err := gw.service.Create(gameID, coordinates)
	if err != nil {
		return err
	}

	gCtx, stop := context.WithCancel(ctx)
	defer stop()
	gameTimer := time.NewTimer(time.Hour)
	defer gameTimer.Stop()
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

				gw.service.Update(g, evt)
				stop()
			case game.GamePlayerNearToTarget:
				gw.service.Update(g, evt)
			case game.GamePlayerAdded, game.GamePlayerRemoved:
				// TODO: monitor game start
				ready := !g.Started() && len(g.Players()) >= MinPlayersPerGame
				if ready {
					gameTimer = time.NewTimer(5 * time.Minute)
					evt = g.Start()
					gw.service.Update(g, evt)
				}
			}
		case <-gameTimer.C:
			log.Printf("GameWorker:watchGame:stop:game:%s", g.ID)
			stop()
		case <-gCtx.Done():
			log.Printf("GameWorker:watchGame:done:game:%s", g.ID)
			evt := g.Stop()
			// TODO store serverID in another store,
			// maybe a env kv store like etcd
			gw.service.Update(g, evt)
			gw.service.Remove(g.ID)
			return nil
		}
	}
}
