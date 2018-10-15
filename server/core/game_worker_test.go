package core_test

import (
	"context"
	"encoding/json"
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
	var playersWithRoles []game.Player
	m.On("Publish", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		event := core.GameEventPayload{}
		json.Unmarshal(data, &event)
		if event.Event == core.GameStarted {
			playersWithRoles = event.Game.Players()
		}
		return true
	})).Return(nil)

	go func() {
		gw := core.NewGameWorker(gs, m)
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
	}()

	completed := make(chan interface{})
	addPlayersToGameServiceMock(gs, g.ID, funk.Values(examplePlayers).([]*game.Player), func() {
		completed <- nil
	})
	<-completed

	assert.Len(t, playersWithRoles, len(examplePlayers))
	for _, p := range playersWithRoles {
		assert.NotEmpty(t, p.Role)
	}

	for _, pl := range examplePlayers {
		p := &core.GameEventPayload{PlayerID: pl.ID, Event: core.GameStarted, Game: g.Game}
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
	example := &core.GameEventPayload{Event: core.GameFinished, Game: g, PlayerID: playerID, DistToTarget: dist}

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

	time.Sleep(time.Second)
	cancel()
	<-complete

	assert.False(t, g.Started())
	gs.AssertCalled(t, "Remove", g.ID)

	p := &core.GameEventPayload{PlayerID: playerID, Event: core.GameFinished,
		Game: game.NewGameWithParams(g.ID, false, nil, "")}
	smocks.AssertPublished(t, m, gameWorkerTopic, p, time.Second)
}

func TestGameWorkerFinishTheGameWhenTimeIsOver(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	core.GameTimeOut = 2 * time.Second

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}

	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)
	m.On("Publish", mock.Anything, mock.Anything).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	players := funk.Values(examplePlayers).([]*game.Player)
	addPlayersToGameServiceMock(gs, g.ID, players,
		func() { assert.Len(t, g.Players(), 3) })

	time.Sleep(core.GameTimeOut + time.Second)
	<-complete

	assert.False(t, g.Started())
	gs.AssertCalled(t, "Remove", g.ID)

	for _, p := range players {
		payload := &core.GameEventPayload{PlayerID: p.ID, Event: core.GameFinished,
			Game: game.NewGameWithParams(g.ID, false, nil, "")}
		smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)
	}
}

func TestGameWorkerStartWithAsPlayersEnterAndNotifyThenThatTheGameStarted(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)

	m.On("Publish", mock.Anything, mock.Anything).Return(nil)

	callbackReached := make(chan func(model.Player, service.GamePlayerAction) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerAction) error) bool {
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
		playerMoveCallback(p.Player, service.GamePlayerActionEnter)
	}

	gamePlayers := g.Players()
	targetID := g.TargetID()
	for _, p := range examplePlayers {
		payload := &core.GameEventPayload{PlayerID: p.ID, Event: core.GameStarted,
			Game: game.NewGameWithParams(g.ID, true, gamePlayers, targetID)}
		smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)
	}
}

func TestGameWorkerFinishTheGameWhenGameIsRunningWhithoutPlayers(t *testing.T) {
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

	callbackReached := make(chan func(model.Player, service.GamePlayerAction) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerAction) error) bool {
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
		playerMoveCallback(p.Player, service.GamePlayerActionEnter)
	}
	<-gameStartedCH
	for _, p := range examplePlayers {
		playerMoveCallback(p.Player, service.GamePlayerActionExit)
	}
	<-complete

	for _, p := range examplePlayers {
		payload := &core.GameEventPayload{PlayerID: p.ID, Event: core.GameFinished,
			Game: game.NewGameWithParams(g.ID, false, nil, "")}
		smocks.AssertPublished(t, m, gameWorkerTopic, payload, time.Second)
	}
}

func TestGameWorkerNotifiesWhenPlayerLose(t *testing.T) {
	m := &smocks.Dispatcher{}
	gs := &smocks.GameService{}
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	players := []game.Player{}
	for _, p := range examplePlayers {
		players = append(players, *p)
	}
	loser := players[0]

	g := game.NewGameWithParams("game-test-1", true, players, players[2].ID)
	gwc := &service.GameWithCoords{Game: g}

	gs.On("Create", mock.Anything, mock.Anything).Return(gwc, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)
	m.On("Publish", mock.Anything, mock.Anything).Return(nil)

	regiserPlayerLose := make(chan interface{})
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerAction) error) bool {
			cb(loser.Player, service.GamePlayerActionExit)
			go func() { regiserPlayerLose <- nil }()
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": gwc.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	<-regiserPlayerLose
	targetID := g.TargetID()
	actualGamePlayers := g.Players()
	actualLoser := funk.Find(actualGamePlayers, func(p game.Player) bool {
		return p.ID == loser.ID
	}).(game.Player)

	cancel()
	<-complete

	assert.True(t, actualLoser.Lose)

	actualPayload := &core.GameEventPayload{Event: core.GamePlayerLose, PlayerID: loser.ID,
		Game: game.NewGameWithParams(g.ID, true, actualGamePlayers, targetID)}
	smocks.AssertPublished(t, m, gameWorkerTopic, actualPayload, time.Second)
}

func addPlayersToGameServiceMock(gs *smocks.GameService, gameID string, players []*game.Player, afterAdd func()) {
	gs.On("ObserveGamePlayers", mock.Anything, gameID,
		mock.MatchedBy(func(cb func(model.Player, service.GamePlayerAction) error) bool {
			for _, p := range players {
				cb(p.Player, service.GamePlayerActionEnter)
			}
			afterAdd()
			return true
		}),
	).Return(nil)
}
