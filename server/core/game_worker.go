package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/service/messages"
	"github.com/perenecabuto/CatchCatch/server/worker"
)

var (
	// ErrGameStoped happens when game can't change anymore
	ErrGameStoped = errors.New("game stoped")
	// ErrGameIDCantBeEmpty happens when run is called without game id
	ErrGameIDCantBeEmpty = errors.New("gameID can't be empty")
	// ErrGameCoordsCantBeEmpty happens when run is called without game id
	ErrGameCoordsCantBeEmpty = errors.New("coords can't be empty")
)

const (
	minPlayersPerGame = 3
	gameChangeTopic   = "game:update"
)

var (
	// GameTimeOut default 5 min
	GameTimeOut = 5 * time.Minute
)

// GameWatcherEvent represents game events for players
type GameWatcherEvent string

func (e GameWatcherEvent) String() string {
	return string(e)
}

// GameWatcherEvent options
const (
	GameStarted            = GameWatcherEvent("game:started")
	GamePlayerNearToTarget = GameWatcherEvent("game:player:near")
	GamePlayerLose         = GameWatcherEvent("game:player:lose")
	GamePlayerWin          = GameWatcherEvent("game:player:win")
	GameFinished           = GameWatcherEvent("game:finished")
)

// GameEventPayload ...
type GameEventPayload struct {
	PlayerID     string
	Game         *game.Game
	Event        GameWatcherEvent
	DistToTarget float64
}

// GameWorker observe manage and run games
type GameWorker struct {
	service  service.GameService
	messages messages.Dispatcher
}

// NewGameWorker creates GameWorker
func NewGameWorker(s service.GameService, m messages.Dispatcher) *GameWorker {
	return &GameWorker{s, m}
}

// ID implementation of worker.Worker.ID()
func (gw GameWorker) ID() string {
	return "GameWorker"
}

// func (gw *GameWorker) OnGameAround(ctx context.Context, cb func(p model.Player, g service.GameWithCoords) error) error {
// 	return nil
// }

// OnGameEvent notifies games events
func (gw *GameWorker) OnGameEvent(ctx context.Context, cb func(payload *GameEventPayload) error) error {
	return gw.messages.Subscribe(ctx, gameChangeTopic, func(data []byte) error {
		payload := &GameEventPayload{}
		err := json.Unmarshal(data, payload)
		// TODO: check better if it will not stop the listener
		if err != nil {
			return err
		}
		return cb(payload)
	})
}

// Run starts this Worker to listen to player events over games
func (gw GameWorker) Run(ctx context.Context, params worker.TaskParams) error {
	gameID, ok := params["gameID"].(string)
	if !ok {
		return ErrGameIDCantBeEmpty
	}
	coordinates, ok := params["coordinates"].(string)
	if !ok {
		return ErrGameCoordsCantBeEmpty
	}

	// FIXME: avoid duplicated games
	// gw.service.Remove(gameID)
	// notify game id to destroy
	// listen to game destroy and exit if this message arrives here
	log.Printf("GameWatcher:create:%s", gameID)
	g, err := gw.service.Create(gameID, coordinates)
	if err != nil {
		return err
	}

	gCtx, stop := context.WithCancel(ctx)
	defer stop()
	evtChan := make(chan game.Event, 1)
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

	gameTimer := time.NewTimer(time.Hour)
	defer gameTimer.Stop()
	for {
		select {
		case evt, ok := <-evtChan:
			if !ok {
				stop()
				break
			}
			started, finished, err :=
				gw.processGameEvent(g, evt)
			if finished || err != nil {
				stop()
				break
			}
			if started {
				// TODO: monitor game start
				gameTimer = time.NewTimer(GameTimeOut)
			}
		case <-gameTimer.C:
			// TODO: notificar Game Timed Out
			log.Printf("GameWorker:watchGame:stop:game:%s", g.ID)
			stop()
		case <-gCtx.Done():
			log.Printf("GameWorker:watchGame:done:game:%s", g.ID)
			stop()
			for _, gp := range g.Players() {
				err := gw.publish(GameFinished, gp, g)
				if err != nil {
					return err
				}
			}
			g.Stop()
			return gw.service.Remove(g.ID)
		}
	}
}

func (gw *GameWorker) publish(evt GameWatcherEvent, gp game.Player, g *service.GameWithCoords) error {
	p := &GameEventPayload{Event: evt, PlayerID: gp.ID, Game: g.Game, DistToTarget: gp.DistToTarget}
	data, _ := json.Marshal(p)
	err := gw.messages.Publish(gameChangeTopic, data)
	if err != nil {
		return fmt.Errorf("GameWorker:watchGame:%s:error:%s - %#v", p.Game.ID, err.Error(), p)
	}
	return nil
}

func (gw *GameWorker) processGameEvent(
	g *service.GameWithCoords, gevt game.Event) (started bool, finished bool, err error) {

	log.Printf("GameWorker:%s:GameEvent:%-v", g.ID, gevt)
	switch gevt.Name {
	case game.GamePlayerNearToTarget:
		gp := gevt.Player
		err = gw.publish(GamePlayerNearToTarget, gp, g)
	case game.GamePlayerAdded, game.GamePlayerRemoved:
		ready := !g.Started() && len(g.Game.Players()) >= minPlayersPerGame
		if ready {
			g.Start()
			started = true

			go gw.service.Update(g)
			for _, gp := range g.Players() {
				err = gw.publish(GameStarted, gp, g)
				if err != nil {
					return false, false, err
				}
			}
			if err != nil {
				err = fmt.Errorf("GameWorker:Start:%s:error:%s - %#v", g.ID, err.Error(), gevt)
			}
		}
	case game.GamePlayerLose:
		err = gw.publish(GamePlayerLose, gevt.Player, g)
	case game.GameTargetWin:
		finished = true
		for _, gp := range g.Players() {
			if gp.Role == game.GameRoleTarget {
				err = gw.publish(GamePlayerWin, gp, g)
			} else {
				err = gw.publish(GamePlayerLose, gp, g)
			}
			if err != nil {
				break
			}
		}
	case game.GameTargetLose:
		finished = true
		target := g.TargetPlayer()
		if target == nil {
			err = errors.New("[GameWorker] target not found")
			break
		}
		err = gw.publish(GamePlayerLose, *target, g)
		if err != nil {
			break
		}
		err = gw.publish(GamePlayerWin, gevt.Player, g)
	case game.GameLastPlayerDetected,
		game.GameRunningWithoutPlayers:
		finished = true
	}
	return started, finished, err
}
