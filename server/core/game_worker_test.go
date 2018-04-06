package core_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/worker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"

	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
)

var (
	gameWorkerTopic = "game:update"
)

func TestGameWorkerStartGame(t *testing.T) {
	ctx, finish := context.WithCancel(context.Background())
	g := &service.GameWithCoords{Game: game.NewGame("test-gameworker-game-1")}

	gs := new(smocks.GameService)
	gs.On("Create", mock.Anything, mock.Anything).Return(g, nil)
	gs.On("Remove", mock.Anything).Return(nil)
	gs.On("Update", mock.Anything).Return(nil)

	examplePlayers := map[string]*game.Player{
		"test-gameworker-player-1": &game.Player{Player: model.Player{ID: "test-gameworker-player-1"}},
		"test-gameworker-player-2": &game.Player{Player: model.Player{ID: "test-gameworker-player-2"}},
		"test-gameworker-player-3": &game.Player{Player: model.Player{ID: "test-gameworker-player-3"}},
	}
	gs.On("ObserveGamePlayers", mock.Anything, g.ID,
		mock.MatchedBy(func(cb func(model.Player, bool) error) bool {
			for _, p := range examplePlayers {
				cb(p.Player, false)
			}
			finish()
			return true
		}),
	).Return(nil)

	m := new(smocks.Dispatcher)
	received := map[string]core.GameEventPayload{}
	m.On("Publish", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		actual := core.GameEventPayload{}
		json.Unmarshal(data, &actual)
		received[actual.PlayerID] = actual
		return true
	})).Return(nil)

	go func() {
		w := core.NewGameWorker(gs, m)
		err := w.Run(ctx, worker.TaskParams{"gameID": g.ID, "coordinates": g.Coords})
		require.NoError(t, err)
	}()

	<-ctx.Done()

	var targetID string
	for _, r := range received {
		targetID = r.Game.TargetID()
		for _, p := range r.Game.Players() {
			examplePlayers[p.ID].Role = p.Role
		}
		break
	}

	exampleList := []game.Player{}
	for _, e := range examplePlayers {
		exampleList = append(exampleList, *e)
	}
	exampleGame := game.NewGameWithParams(g.ID, true, exampleList, targetID)
	examples := map[string]core.GameEventPayload{}
	for _, p := range exampleGame.Players() {
		payload := core.GameEventPayload{Event: core.GameStarted,
			Game:         exampleGame,
			PlayerID:     p.ID,
			DistToTarget: p.DistToTarget}
		examples[p.ID] = payload
	}

	m.AssertCalled(t, "Publish", gameWorkerTopic, mock.MatchedBy(func([]byte) bool {
		jsonE, _ := json.Marshal(examples)
		jsonR, _ := json.Marshal(received)
		return assert.JSONEq(t, string(jsonE), string(jsonR))
	}))
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
