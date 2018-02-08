package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"

	"github.com/perenecabuto/CatchCatch/server/mocks/repo_mocks"
)

var (
	gameID   = "test-game-service-game1"
	serverID = "test-game-service-server1"
)

func TestGameServiceCreate(t *testing.T) {
	repo := &repo_mocks.Repository{}
	stream := &repo_mocks.EventStream{}
	service := NewGameService(repo, stream)

	gameFeat := &model.Feature{ID: gameID, Coordinates: ""}
	repo.On("FeatureByID", "geofences", gameID).Return(gameFeat, nil)
	repo.On("SetFeature", "game", gameID, gameFeat.Coordinates).
		Return(gameFeat, nil)
	repo.On("SetFeatureExtraData", "game", gameID, mock.Anything).
		Return(nil)

	now := time.Now()
	service.Create(gameID, serverID)

	matchPayload := mock.MatchedBy(func(payload string) bool {
		gameEvt := GameEvent{}
		json.Unmarshal([]byte(payload), &gameEvt)

		return assert.Equal(t, gameEvt.Event, game.GameEventCreated) &&
			assert.Equal(t, serverID, gameEvt.ServerID) &&
			assertDateEqual(t, now, gameEvt.LastUpdate)
	})
	repo.AssertCalled(t, "SetFeatureExtraData", "game",
		gameID, matchPayload)
}

func TestGameServiceMustGetNewGame(t *testing.T) {
	repo := &repo_mocks.Repository{}
	stream := &repo_mocks.EventStream{}
	service := NewGameService(repo, stream)

	players := make([]game.Player, 0)
	expectedGame := game.NewGameWithParams(gameID, false, players, "")

	gameEvt := GameEvent{Game: *expectedGame, Event: game.GameEventNothing}
	serialized, _ := json.Marshal(gameEvt)
	repo.On("FeatureExtraData", "game", gameID).Return(string(serialized), nil)

	game, evt, err := service.GameByID(gameID)
	assert.NoError(t, err)
	assert.Equal(t, expectedGame, game)
	assert.NotNil(t, evt)
}

func TestGameServiceMustGetGameWithPlayers(t *testing.T) {
	repo := &repo_mocks.Repository{}
	stream := &repo_mocks.EventStream{}
	service := NewGameService(repo, stream)

	players := []game.Player{
		game.Player{Player: model.Player{ID: "player-1"}, Role: game.GameRoleHunter},
		game.Player{Player: model.Player{ID: "player-2"}, Role: game.GameRoleHunter},
		game.Player{Player: model.Player{ID: "player-3"}, Role: game.GameRoleTarget},
	}
	expectedGame := game.NewGameWithParams(gameID, true, players, "player-3")

	gameEvt := GameEvent{Game: *expectedGame, Event: game.GameEventNothing}
	serialized, _ := json.Marshal(gameEvt)
	repo.On("FeatureExtraData", "game", gameID).Return(string(serialized), nil)

	game, evt, err := service.GameByID(gameID)

	assert.NoError(t, err)
	assert.Equal(t, expectedGame, game)
	assert.Equal(t, gameEvt.Event, *evt)
}

func assertDateEqual(t *testing.T, date1, date2 time.Time) bool {
	return assert.Condition(t, func() bool {
		return assert.Equal(t, date1.Day(), date2.Day()) &&
			assert.Equal(t, date1.Month(), date2.Month()) &&
			assert.Equal(t, date1.Year(), date2.Year())
	})
}
