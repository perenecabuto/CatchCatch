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
	m := new(smocks.Dispatcher)
	gs := new(smocks.GameService)
	gw := core.NewGameWorker(gs, m)
	ctx := context.Background()

	err := gw.Run(ctx, worker.TaskParams{})

	assert.EqualError(t, err, core.ErrGameIDCantBeEmpty.Error())
}

func TestGameWorkerDoNotRunWithoutCoords(t *testing.T) {
	m := new(smocks.Dispatcher)
	gs := new(smocks.GameService)
	gw := core.NewGameWorker(gs, m)
	ctx := context.Background()

	err := gw.Run(ctx, worker.TaskParams{"gameID": "test-game-1"})

	assert.EqualError(t, err, core.ErrGameCoordsCantBeEmpty.Error())
}

func TestGameWorkerStartsWhenTheNumberOfPlayersIsEnough(t *testing.T) {
	ctx := context.Background()
	g := &service.GameWithCoords{Game: game.NewGame("test-gameworker-game-1")}
	gs := new(smocks.GameService)
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Remove", mock.Anything).Return(nil)
	gs.On("Update", mock.Anything).Return(nil)

	completed := make(chan interface{})
	addPlayersToGameServiceMock(gs, g.ID, funk.Values(examplePlayers).([]*game.Player), func() {
		completed <- nil
	})

	m := new(smocks.Dispatcher)
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

	<-completed

	assert.Len(t, playersWithRoles, len(examplePlayers))
	for _, p := range playersWithRoles {
		assert.NotEmpty(t, p.Role)
	}

	smocks.AssertPublished(t, m, time.Second, gameWorkerTopic, func(data []byte) bool {
		p := &core.GameEventPayload{}
		json.Unmarshal(data, p)
		return p.Event == core.GameStarted
	})
}

func TestGameWorkerMustObserveGameChangeEvents(t *testing.T) {
	m := new(smocks.Dispatcher)
	gs := new(smocks.GameService)
	gw := core.NewGameWorker(gs, m)
	ctx := context.Background()

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
	m := new(smocks.Dispatcher)
	gs := new(smocks.GameService)
	gw := core.NewGameWorker(gs, m)
	ctx, cancel := context.WithCancel(context.Background())

	playerID := "game-test-1-player-1"

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}
	g.Game.SetPlayer(playerID, 0, 0)
	g.Start()
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("ObserveGamePlayers", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)
	m.On("Publish", mock.Anything, mock.Anything).Return(nil)

	complete := make(chan interface{})
	go func() {
		gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		complete <- nil
	}()

	time.Sleep(time.Second)
	cancel()
	<-complete

	assert.False(t, g.Started())
	gs.AssertCalled(t, "Remove", g.ID)
	smocks.AssertPublished(t, m, time.Second, gameWorkerTopic, func(data []byte) bool {
		p := &core.GameEventPayload{}
		json.Unmarshal(data, p)
		return p.Event == core.GameFinished
	})
}

func TestGameWorkerFinishTheGameWhenTimeIsOver(t *testing.T) {
	m := new(smocks.Dispatcher)
	gs := new(smocks.GameService)
	gw := core.NewGameWorker(gs, m)
	ctx := context.Background()

	core.GameTimeOut = 2 * time.Second

	g := &service.GameWithCoords{Game: game.NewGame("game-test-1")}
	players := funk.Values(examplePlayers).([]*game.Player)
	addPlayersToGameServiceMock(gs, g.ID, players, func() {
		assert.Len(t, g.Players(), 3)
	})

	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Update", mock.Anything).Return(nil)
	gs.On("ObserveGamePlayers", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	gs.On("Remove", mock.Anything).Return(nil)
	m.On("Publish", mock.Anything, mock.Anything).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	time.Sleep(core.GameTimeOut + time.Second)

	assert.False(t, g.Started())
	gs.AssertCalled(t, "Remove", g.ID)

	<-complete
	smocks.AssertPublished(t, m, time.Second, gameWorkerTopic, func(data []byte) bool {
		p := &core.GameEventPayload{}
		json.Unmarshal(data, p)
		return p.Event == core.GameFinished
	})
}

func TestGameWorkerFinishTheGameWhenGameIsRunningWhithoutPlayers(t *testing.T) {
	m := new(smocks.Dispatcher)
	gs := new(smocks.GameService)
	gw := core.NewGameWorker(gs, m)
	ctx := context.Background()

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

	callbackReached := make(chan func(model.Player, bool) error)
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, bool) error) bool {
			callbackReached <- cb
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	callback := <-callbackReached
	for _, p := range examplePlayers {
		callback(p.Player, false)
	}
	<-gameStartedCH
	for _, p := range examplePlayers {
		callback(p.Player, true)
	}
	<-complete

	smocks.AssertPublished(t, m, time.Second, gameWorkerTopic, func(data []byte) bool {
		p := &core.GameEventPayload{}
		json.Unmarshal(data, p)
		return p.Event == core.GameFinished
	})
}

func TestGameWorkerNotifiesWhenPlayerLose(t *testing.T) {
	m := new(smocks.Dispatcher)
	gs := new(smocks.GameService)
	gw := core.NewGameWorker(gs, m)
	ctx, finish := context.WithCancel(context.Background())

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
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, bool) error) bool {
			cb(loser.Player, true)
			finish()
			return true
		}),
	).Return(nil)

	complete := make(chan interface{})
	go func() {
		err := gw.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": gwc.Coords})
		require.NoError(t, err)
		complete <- nil
	}()

	<-complete
	smocks.AssertPublished(t, m, time.Second, gameWorkerTopic, func(data []byte) bool {
		p := &core.GameEventPayload{}
		json.Unmarshal(data, p)
		return p.Event == core.GamePlayerLose && p.PlayerID == loser.ID
	})
}

func addPlayersToGameServiceMock(gs *smocks.GameService, gameID string, players []*game.Player, afterAdd func()) {
	gs.On("ObserveGamePlayers", mock.Anything, gameID,
		mock.MatchedBy(func(cb func(model.Player, bool) error) bool {
			for _, p := range players {
				cb(p.Player, false)
			}
			afterAdd()
			return true
		}),
	).Return(nil)
}
