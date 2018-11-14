package core_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	funk "github.com/thoas/go-funk"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/worker"

	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
)

var (
	gameWorkerTopic = "game:update"

	examplePlayers = map[string]*game.Player{
		"test-gameworker-player-1": &game.Player{Player: model.Player{ID: "test-gameworker-player-1"}},
		"test-gameworker-player-2": &game.Player{Player: model.Player{ID: "test-gameworker-player-2"}},
		"test-gameworker-player-3": &game.Player{Player: model.Player{ID: "test-gameworker-player-3"}},
	}
)

func TestGameWorkerDoNotRunWithoutGameID(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := gw.Run(ctx, worker.TaskParams{})

	assert.EqualError(t, err, core.ErrGameIDCantBeEmpty.Error())
}

func TestGameWorkerDoNotRunWithoutCoords(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := gw.Run(ctx, worker.TaskParams{"gameID": "test-game-1"})

	assert.EqualError(t, err, core.ErrGameCoordsCantBeEmpty.Error())
}

func TestGameWorkerStartsWhenTheNumberOfPlayersIsEnough(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := &service.GameWithCoords{Game: game.NewGame("test-gameworker-game-1")}
	gs := &smocks.GameService{}
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Remove", mock.Anything).Return(nil)
	gs.On("Update", mock.Anything).Return(nil)

	m := &smocks.Dispatcher{}
	gameStartedCH := make(chan interface{})
	m.On("Publish", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		event := core.GameEventPayload{}
		json.Unmarshal(data, &event)
		if event.Event == core.GameStarted {
			go func() { gameStartedCH <- nil }()
		}
		return true
	})).Return(nil)

	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerMove) error) bool {
			for _, p := range examplePlayers {
				cb(p.Player, service.GamePlayerMoveInside)
			}
			return true
		}),
	).Return(nil)

	completed := make(chan interface{})
	go func() {
		gw := core.NewGameWorker(gs, m)
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
		completed <- nil
	}()

	<-gameStartedCH
	gamePlayers := g.Players()

	cancel()
	time.Sleep(time.Millisecond * 100)
	<-completed

	assert.Len(t, gamePlayers, len(examplePlayers))
	for _, p := range gamePlayers {
		assert.NotEmpty(t, p.Role)
	}
	for _, pl := range gamePlayers {
		p := &core.GameEventPayload{PlayerID: pl.ID, PlayerRole: pl.Role,
			Event: core.GameStarted, Game: g.Game.ID}
		smocks.AssertPublished(t, m, gameWorkerTopic, p, time.Second)
	}
}

func TestGameWorkerMustObserveGameChangeEvents(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := game.NewGame("test-game-1")
	playerID := "test-game-player-1"
	dist := 100.0
	example := &core.GameEventPayload{Event: core.GameFinished, Game: g.ID, PlayerID: playerID, DistToTarget: dist}

	m.On("Subscribe", mock.Anything, mock.Anything,
		mock.MatchedBy(func(cb func(data []byte) error) bool {
			data, _ := json.Marshal(example)
			cb(data)
			return true
		})).Return(nil)

	var actual *core.GameEventPayload
	gw.OnGameEvent(ctx, func(p *core.GameEventPayload) error {
		actual = p
		return nil
	})

	assert.Equal(t, example, actual)
}

func TestGameWorkerFinishTheGameWhenContextIsDone(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())

	playerID := "game-test-1-player-1"

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("ObserveGamePlayers", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)
	m.On("Publish", mock.Anything, mock.Anything).Return(nil)

	complete := make(chan interface{})
	go func() {
		gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		complete <- nil
	}()

	g.SetPlayer(playerID, 0, 0)
	g.Start()

	player := g.Players()[0]
	rank := g.Rank()

	cancel()
	<-complete

	assert.False(t, g.Started())
	gs.AssertCalled(t, "Remove", g.ID)

	p := &core.GameEventPayload{PlayerID: player.ID, PlayerRole: player.Role,
		Event: core.GameFinished, Game: g.ID, Rank: rank}
	smocks.AssertPublished(t, m, gameWorkerTopic, p, time.Second)
}

func TestGameWorkerDoNotSendFinishMessageWhenItStopWithAnNotStartedGame(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}
	gs.On("Remove", mock.Anything).Return(nil)
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("ObserveGamePlayers", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	complete := make(chan interface{})
	go func() {
		gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		complete <- nil
	}()

	cancel()
	<-complete

	assert.False(t, g.Started())
	gs.AssertCalled(t, "Remove", g.ID)

	m.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
}

func TestGameWorkerFinishWhenTimeIsOverAndTargetWinsTheStartedGame(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	core.GameTimeOut = time.Millisecond * 100

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}

	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)

	gameStartedCH := make(chan interface{})
	m.On("Publish", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		event := core.GameEventPayload{}
		json.Unmarshal(data, &event)
		if event.Event == core.GameStarted {
			go func() { gameStartedCH <- nil }()
		}
		return true
	})).Return(nil)

	callbackReached := make(chan func(model.Player, service.GamePlayerMove) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerMove) error) bool {
			go func() { callbackReached <- cb }()
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	playerMoveCallback := <-callbackReached
	for _, p := range examplePlayers {
		playerMoveCallback(p.Player, service.GamePlayerMoveInside)
	}

	<-gameStartedCH
	gamePlayers := g.Players()
	rank := game.NewGameRank(g.ID).
		ByPlayersDistanceToTarget(funk.Map(gamePlayers, func(p game.Player) game.Player {
			p.Lose = p.Role == game.GameRoleHunter
			return p
		}).([]game.Player))

	time.Sleep(core.GameTimeOut + (time.Millisecond * 100))
	<-complete

	assert.False(t, g.Started())
	gs.AssertCalled(t, "Remove", g.ID)

	for _, p := range gamePlayers {
		payload := &core.GameEventPayload{PlayerID: p.ID, PlayerRole: p.Role,
			Event: core.GameFinished, Game: g.ID, Rank: rank}
		smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)
	}
}

func TestGameWorkerJustFinishWhenTimeIsOverAndTheGameWasNotStarted(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	core.GameWorkerIdleTimeOut = time.Millisecond * 100

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}

	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)
	m.On("Publish", mock.Anything, mock.Anything).Return(nil)

	callbackReached := make(
		chan func(model.Player, service.GamePlayerMove) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerMove) error) bool {
			go func() { callbackReached <- cb }()
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	playerMoveCallback := <-callbackReached
	exP := examplePlayers["test-gameworker-player-1"]
	playerMoveCallback(exP.Player, service.GamePlayerMoveInside)

	time.Sleep(core.GameWorkerIdleTimeOut + (time.Millisecond * 100))
	<-complete

	assert.False(t, g.Started())
	gs.AssertCalled(t, "Remove", g.ID)

	m.AssertNotCalled(t, "Publish")
}

func TestGameWorkerStartsAsPlayersEnterAndNotifyThenThatTheGameStarted(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)

	gameStartedCH := make(chan interface{})
	m.On("Publish", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		event := core.GameEventPayload{}
		json.Unmarshal(data, &event)
		if event.Event == core.GameStarted {
			go func() { gameStartedCH <- nil }()
		}
		return true
	})).Return(nil)

	callbackReached := make(chan func(model.Player, service.GamePlayerMove) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerMove) error) bool {
			go func() { callbackReached <- cb }()
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	playerMoveCallback := <-callbackReached
	for _, p := range examplePlayers {
		playerMoveCallback(p.Player, service.GamePlayerMoveInside)
	}

	<-gameStartedCH
	gamePlayers := g.Players()
	rank := g.Rank()

	cancel()
	<-complete

	for _, p := range gamePlayers {
		payload := &core.GameEventPayload{PlayerID: p.ID, PlayerRole: p.Role,
			Event: core.GameFinished, Game: g.ID, Rank: rank}
		smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)
	}
}

func TestGameWorkerFinishTheGameWhenGameIsRunningWithoutPlayers(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)

	gameStartedCH := make(chan interface{})
	m.On("Publish", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		event := core.GameEventPayload{}
		json.Unmarshal(data, &event)
		if event.Event == core.GameStarted {
			go func() { gameStartedCH <- nil }()
		}
		return true
	})).Return(nil)

	callbackReached := make(chan func(model.Player, service.GamePlayerMove) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerMove) error) bool {
			go func() { callbackReached <- cb }()
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	playerMoveCallback := <-callbackReached
	for _, p := range examplePlayers {
		playerMoveCallback(p.Player, service.GamePlayerMoveInside)
	}
	<-gameStartedCH
	gamePlayers := g.Players()

	for _, p := range examplePlayers {
		playerMoveCallback(p.Player, service.GamePlayerMoveOutside)
	}

	rank := game.NewGameRank(g.ID).
		ByPlayersDistanceToTarget(funk.Map(gamePlayers, func(p game.Player) game.Player {
			p.Lose = true
			return p
		}).([]game.Player))

	<-complete
	for _, p := range gamePlayers {
		payload := &core.GameEventPayload{PlayerID: p.ID, PlayerRole: p.Role,
			Event: core.GameFinished, Game: g.ID, Rank: rank}
		smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)
	}
}

func TestGameWorkerNotifiesWhenLastPlayerIsInGame(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)

	gameStartedCH := make(chan interface{})
	m.On("Publish", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		event := core.GameEventPayload{}
		json.Unmarshal(data, &event)
		if event.Event == core.GameStarted {
			go func() { gameStartedCH <- nil }()
		}
		return true
	})).Return(nil)

	callbackReached := make(chan func(model.Player, service.GamePlayerMove) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerMove) error) bool {
			go func() { callbackReached <- cb }()
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	playerMoveCallback := <-callbackReached
	for _, p := range examplePlayers {
		playerMoveCallback(p.Player, service.GamePlayerMoveInside)
	}

	<-gameStartedCH
	gamePlayers := g.Players()
	target := funk.Find(gamePlayers, func(p game.Player) bool {
		return p.Role == game.GameRoleTarget
	}).(game.Player)
	for _, p := range gamePlayers {
		if p.Role == game.GameRoleHunter {
			playerMoveCallback(p.Player, service.GamePlayerMoveOutside)
		}
	}
	<-complete

	payload := &core.GameEventPayload{
		PlayerID: target.ID, PlayerRole: game.GameRoleTarget,
		Event: core.GamePlayerWin, Game: g.ID}
	smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)
}

func TestGameWorkerNotifiesFinishWhenTargetLeaveTheGame(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)

	gameStartedCH := make(chan interface{})
	m.On("Publish", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		event := core.GameEventPayload{}
		json.Unmarshal(data, &event)
		if event.Event == core.GameStarted {
			go func() { gameStartedCH <- nil }()
		}
		return true
	})).Return(nil)

	callbackReached := make(chan func(model.Player, service.GamePlayerMove) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerMove) error) bool {
			go func() { callbackReached <- cb }()
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	playerMoveCallback := <-callbackReached
	for _, p := range examplePlayers {
		playerMoveCallback(p.Player, service.GamePlayerMoveInside)
	}

	<-gameStartedCH
	gamePlayers := g.Players()
	target := funk.Find(gamePlayers, func(p game.Player) bool {
		return p.Role == game.GameRoleTarget
	}).(game.Player)

	playerMoveCallback(target.Player, service.GamePlayerMoveOutside)
	rank := g.Rank()

	<-complete

	payload := &core.GameEventPayload{
		PlayerID: target.ID, PlayerRole: game.GameRoleTarget,
		Event: core.GamePlayerLose, Game: g.ID}
	smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)

	for _, p := range gamePlayers {
		payload := &core.GameEventPayload{PlayerID: p.ID, PlayerRole: p.Role,
			Event: core.GameFinished, Game: g.ID, Rank: rank}
		smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)
	}
}

func TestGameWorkerNotifiesWhenTargetIsReached(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)

	gameStartedCH := make(chan interface{})
	m.On("Publish", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		event := core.GameEventPayload{}
		json.Unmarshal(data, &event)
		if event.Event == core.GameStarted {
			go func() { gameStartedCH <- nil }()
		}
		return true
	})).Return(nil)

	callbackReached := make(chan func(model.Player, service.GamePlayerMove) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerMove) error) bool {
			go func() { callbackReached <- cb }()
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	playerMoveCallback := <-callbackReached
	for i := 0; i < 3; i++ {
		p := game.Player{Player: model.Player{ID: "player-" + strconv.Itoa(i),
			Lat: float64(i), Lon: float64(i)}}
		playerMoveCallback(p.Player, service.GamePlayerMoveInside)
	}

	<-gameStartedCH
	target := funk.Find(g.Players(), func(p game.Player) bool {
		return p.Role == game.GameRoleTarget
	}).(game.Player)
	hunter := funk.Find(g.Players(), func(p game.Player) bool {
		return p.Role == game.GameRoleHunter
	}).(game.Player)

	hunter.Player.Lat, hunter.Player.Lon =
		target.Lat+0.00001, target.Lon+0.00001
	playerMoveCallback(hunter.Player, service.GamePlayerMoveInside)

	gamePlayers := g.Players()
	rank := g.Rank()
	<-complete

	payloads := []core.GameEventPayload{
		{
			PlayerID: hunter.ID, PlayerRole: game.GameRoleHunter,
			Event: core.GamePlayerWin, Game: g.ID,
			DistToTarget: hunter.DistTo(target.Player),
		},
		{
			PlayerID: target.ID, PlayerRole: game.GameRoleTarget,
			Event: core.GamePlayerLose, Game: g.ID,
			DistToTarget: 0,
		},
	}
	for _, p := range gamePlayers {
		payloads = append(payloads, core.GameEventPayload{
			PlayerID: p.ID, PlayerRole: p.Role,
			Event: core.GameFinished, Game: g.ID, Rank: rank,
		})
	}
	for _, p := range payloads {
		smocks.AssertPublished(t, m, gameWorkerTopic, &p, time.Second)
	}
}

func TestGameWorkerNotifiesWhenTargetWinsWhileInArenaAfterTimeout(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := game.NewGame("game-test-1")
	gwc := &service.GameWithCoords{Game: g}

	gs.On("Create", mock.Anything, mock.Anything).Return(gwc, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)

	gameStartedCH := make(chan interface{})
	m.On("Publish", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		event := core.GameEventPayload{}
		json.Unmarshal(data, &event)
		if event.Event == core.GameStarted {
			go func() { gameStartedCH <- nil }()
		}
		return true
	})).Return(nil)

	callbackReached := make(chan func(model.Player, service.GamePlayerMove) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerMove) error) bool {
			go func() { callbackReached <- cb }()
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": gwc.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	core.GameTimeOut = time.Millisecond * 100

	playerMoveCallback := <-callbackReached
	for i := 0; i < 3; i++ {
		p := game.Player{Player: model.Player{ID: "player-" + strconv.Itoa(i),
			Lat: float64(i), Lon: float64(i)}}
		playerMoveCallback(p.Player, service.GamePlayerMoveInside)
	}

	<-gameStartedCH

	gamePlayers := g.Players()
	target := funk.Find(gamePlayers, func(p game.Player) bool {
		return p.Role == game.GameRoleTarget
	}).(game.Player)

	<-complete
	hunters := []game.Player{}
	for i, p := range gamePlayers {
		if p.Role == game.GameRoleHunter {
			gamePlayers[i].Lose = true
			hunters = append(hunters, p)
		}
	}

	payload := &core.GameEventPayload{Event: core.GamePlayerWin, Game: g.ID,
		PlayerID: target.ID, PlayerRole: game.GameRoleTarget}
	smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)

	for _, p := range hunters {
		payload := &core.GameEventPayload{
			Event:    core.GamePlayerLose,
			PlayerID: p.ID, PlayerRole: game.GameRoleHunter,
			Game: g.ID, DistToTarget: p.DistToTarget}
		smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)
	}
}

func TestGameWorkerNotifiesWhenPlayerLose(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := game.NewGame("game-test-1")
	gwc := &service.GameWithCoords{Game: g}

	gs.On("Create", mock.Anything, mock.Anything).Return(gwc, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)

	gameStartedCH := make(chan interface{})
	m.On("Publish", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		event := core.GameEventPayload{}
		json.Unmarshal(data, &event)
		if event.Event == core.GameStarted {
			go func() { gameStartedCH <- nil }()
		}
		return true
	})).Return(nil)

	callbackReached := make(chan func(model.Player, service.GamePlayerMove) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerMove) error) bool {
			go func() { callbackReached <- cb }()
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": gwc.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	playerMoveCallback := <-callbackReached
	for i := 0; i < 3; i++ {
		p := game.Player{Player: model.Player{ID: "player-" + strconv.Itoa(i),
			Lat: float64(i), Lon: float64(i)}}
		playerMoveCallback(p.Player, service.GamePlayerMoveInside)
	}

	<-gameStartedCH
	loser := g.Players()[0]
	playerMoveCallback(loser.Player, service.GamePlayerMoveOutside)

	gamePlayers := g.Players()
	actualLoser := funk.Find(gamePlayers, func(p game.Player) bool {
		return p.ID == loser.ID
	}).(game.Player)

	time.Sleep(time.Millisecond * 100)
	cancel()
	<-complete

	assert.True(t, actualLoser.Lose)

	payload := &core.GameEventPayload{Event: core.GamePlayerLose,
		PlayerID: loser.ID, PlayerRole: loser.Role,
		Game: g.ID, DistToTarget: loser.DistToTarget}
	smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)
}

func TestGameWorkerNotifiesWhenHunterIsNearToTarget(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)

	gameStartedCH := make(chan interface{})
	m.On("Publish", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		event := core.GameEventPayload{}
		json.Unmarshal(data, &event)
		if event.Event == core.GameStarted {
			go func() { gameStartedCH <- nil }()
		}
		return true
	})).Return(nil)

	callbackReached := make(chan func(model.Player, service.GamePlayerMove) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerMove) error) bool {
			go func() { callbackReached <- cb }()
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	playerMoveCallback := <-callbackReached
	for i := 0; i < 3; i++ {
		p := game.Player{Player: model.Player{ID: "player-" + strconv.Itoa(i),
			Lat: float64(i), Lon: float64(i)}}
		playerMoveCallback(p.Player, service.GamePlayerMoveInside)
	}

	<-gameStartedCH
	target := funk.Find(g.Players(), func(p game.Player) bool {
		return p.Role == game.GameRoleTarget
	}).(game.Player)
	hunter := funk.Find(g.Players(), func(p game.Player) bool {
		return p.Role == game.GameRoleHunter
	}).(game.Player)

	target.Player.Lat, target.Player.Lon = 0, 0
	playerMoveCallback(target.Player, service.GamePlayerMoveInside)
	hunter.Player.Lat, hunter.Player.Lon = 0.0002, 0.0002
	playerMoveCallback(hunter.Player, service.GamePlayerMoveInside)

	time.Sleep(time.Millisecond * 100)
	cancel()
	<-complete

	payload := &core.GameEventPayload{Event: core.GamePlayerNearToTarget, Game: g.ID,
		PlayerID: hunter.ID, PlayerRole: game.GameRoleHunter,
		DistToTarget: 31.45067466553135}
	smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)
}
