package core

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/pkg/errors"

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
	ErrGameIDCantBeEmpty = errors.New("gameID is empty or invalid")
	// ErrGameCoordsCantBeEmpty happens when run is called without game id
	ErrGameCoordsCantBeEmpty = errors.New("coordinates is empty or invalid")
)

const (
	minPlayersPerGame = 3
	gameChangeTopic   = "game:update"
)

var (
	// GameTimeOut default 5 min
	// TODO: move it to worker constructor!!!
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
	PlayerRole   game.Role
	DistToTarget float64
	Game         string
	Rank         game.Rank
	Event        GameWatcherEvent
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
// TODO: monitor errors
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

	ctx, stop := context.WithCancel(ctx)
	defer stop()

	evtChan := gw.observeGameEvents(ctx, g.Game)
	gameTimer := time.NewTimer(time.Hour)
	defer gameTimer.Stop()
	for {
		select {
		case evt, ok := <-evtChan:
			if !ok {
				stop()
				break
			}
			started, finished, err := gw.processGameEvent(g, evt)
			if err != nil {
				// TODO: fix it by worker manager retry
				return err
			}
			if finished {
				stop()
				break
			}
			if started {
				// TODO: monitor game start
				gameTimer = time.NewTimer(GameTimeOut)
			}
		case <-gameTimer.C:
			if g.Started() {
				for _, gp := range g.Players() {
					if gp.Role != game.GameRoleTarget {
						g.RemovePlayer(gp.Player.ID)
					}
				}
				for _, gp := range g.Players() {
					if gp.Role == game.GameRoleTarget {
						gw.publish(GamePlayerWin, &gp, g.Game)
					} else {
						gw.publish(GamePlayerLose, &gp, g.Game)
					}
				}
				log.Printf("GameWorker:watchGame:stop:game:%s", g.ID)
			}
			stop()
		case <-ctx.Done():
			log.Printf("GameWorker:watchGame:done:game:%s", g.ID)
			players := g.Players()
			for _, gp := range players {
				gp.DistToTarget = 0
				err := gw.publish(GameFinished, &gp, g.Game)
				if err != nil {
					return errors.Cause(err)
				}
			}
			g.Stop()
			return errors.Wrapf(gw.service.Remove(g.ID),
				"can't remove game %s", g.ID,
			)
		}
	}
}

func (gw *GameWorker) processGameEvent(g *service.GameWithCoords, gevt game.Event) (
	started bool, finished bool, err error) {

	log.Printf("GameWorker:%s:GameEvent:%-v", g.ID, gevt)
	switch gevt.Name {
	case game.GamePlayerNearToTarget:
		err = gw.publish(GamePlayerNearToTarget, &gevt.Player, g.Game)
	case game.GamePlayerAdded, game.GamePlayerRemoved:
		ready := !g.Started() && len(g.Players()) >= minPlayersPerGame
		if ready {
			g.Start()
			started = true

			go gw.service.Update(g)
			for _, gp := range g.Players() {
				err = gw.publish(GameStarted, &gp, g.Game)
				if err != nil {
					return false, false, err
				}
			}
			if err != nil {
				err = errors.Wrapf(err, "GameWorker:Start:%s:%+v", g.ID, gevt)
			}
		}
	case game.GamePlayerRanWay:
		if gevt.Player.Role == game.GameRoleTarget {
			finished = true
		}
		err = gw.publish(GamePlayerLose, &gevt.Player, g.Game)
		if err != nil {
			break
		}
	case game.GameTargetReached:
		finished = true
		err = gw.publish(GamePlayerWin, &gevt.Player, g.Game)
		if err != nil {
			break
		}
		err = gw.publish(GamePlayerLose, g.TargetPlayer(), g.Game)
		if err != nil {
			break
		}
	case game.GameLastPlayerDetected:
		finished = true
		err = gw.publish(GamePlayerWin, &gevt.Player, g.Game)
	case game.GameRunningWithoutPlayers:
		finished = true
	}
	return started, finished, errors.Wrapf(err, "can't process event %+v", gevt)
}

func (gw *GameWorker) publish(evt GameWatcherEvent, player *game.Player, g *game.Game) error {
	p := &GameEventPayload{
		Event:        evt,
		PlayerID:     player.ID,
		PlayerRole:   player.Role,
		Game:         g.ID,
		DistToTarget: player.DistToTarget,
	}
	if evt == GameFinished {
		p.Rank = g.Rank()
	}
	data, _ := json.Marshal(p)
	err := gw.messages.Publish(gameChangeTopic, data)
	return errors.Wrapf(err, "GameWorker:publish:game:%s:player:%+v", g.ID, p)
}

func (gw *GameWorker) observeGameEvents(ctx context.Context, g *game.Game) chan game.Event {
	evtChan := make(chan game.Event, 1)
	go func() {
		err := gw.service.ObserveGamePlayers(ctx, g.ID, func(p model.Player, a service.GamePlayerMove) error {
			var evt game.Event
			switch a {
			case service.GamePlayerMoveOutside:
				evt = g.RemovePlayer(p.ID)
			case service.GamePlayerMoveInside:
				evt = g.SetPlayer(p.ID, p.Lat, p.Lon)
			}
			if evt.Name != game.GameNothingHappens {
				select {
				case evtChan <- evt:
				case <-ctx.Done():
				}
			}
			return nil
		})
		if err != nil {
			log.Println("Error on ObserveGamePlayers", err)
			close(evtChan)
		}
	}()
	return evtChan
}
