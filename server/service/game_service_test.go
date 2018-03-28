package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"

	"github.com/perenecabuto/CatchCatch/server/mocks/messages_mocks"
	"github.com/perenecabuto/CatchCatch/server/mocks/repo_mocks"
)

var (
	gameID   = "test-game-service-game1"
	serverID = "test-game-service-server1"
)

func TestGameServiceCreate(t *testing.T) {
	repo := &repo_mocks.Repository{}
	stream := &repo_mocks.EventStream{}
	dispatcher := &messages_mocks.Dispatcher{}
	service := NewGameService(repo, stream, dispatcher)

	gameFeat := &model.Feature{ID: gameID, Coordinates: ""}
	repo.On("FeatureByID", "geofences", gameID).Return(gameFeat, nil)
	repo.On("SetFeature", "game", gameID, gameFeat.Coordinates).
		Return(gameFeat, nil)
	repo.On("SetFeatureExtraData", "game", gameID, mock.Anything).
		Return(nil)

	dispatcher.On("Publish", GameChangeTopic, mock.Anything).Return(nil)

	service.Create(gameID, serverID)

	dispatcher.AssertCalled(t, "Publish", GameChangeTopic, matchGameChangePayload(t))
	repo.AssertCalled(t, "SetFeatureExtraData", "game", gameID, matchGameChangePayload(t))
}

func TestGameServiceUpdate(t *testing.T) {
	repo := &repo_mocks.Repository{}
	stream := &repo_mocks.EventStream{}
	dispatcher := &messages_mocks.Dispatcher{}
	service := NewGameService(repo, stream, dispatcher)

	gameFeat := &model.Feature{ID: gameID, Coordinates: ""}
	repo.On("FeatureByID", "geofences", gameID).Return(gameFeat, nil)
	repo.On("SetFeature", "game", gameID, gameFeat.Coordinates).
		Return(gameFeat, nil)
	repo.On("SetFeatureExtraData", "game", gameID, mock.Anything).
		Return(nil)

	dispatcher.On("Publish", GameChangeTopic, mock.Anything).Return(nil)

	g, evt := game.NewGame(gameID)
	service.Update(g, serverID, evt)

	dispatcher.AssertCalled(t, "Publish", GameChangeTopic, matchGameChangePayload(t))
	repo.AssertCalled(t, "SetFeatureExtraData", "game", gameID, matchGameChangePayload(t))
}

func TestGameServiceMustGetNewGame(t *testing.T) {
	repo := &repo_mocks.Repository{}
	stream := &repo_mocks.EventStream{}
	dispatcher := &messages_mocks.Dispatcher{}
	service := NewGameService(repo, stream, dispatcher)

	players := make([]game.Player, 0)
	expectedGame := game.NewGameWithParams(gameID, false, players, "")

	gameEvt := GameEvent{Game: expectedGame, Event: game.GameEventNothing}
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
	dispatcher := &messages_mocks.Dispatcher{}
	service := NewGameService(repo, stream, dispatcher)

	players := []game.Player{
		game.Player{Player: model.Player{ID: "player-1"}, Role: game.GameRoleHunter},
		game.Player{Player: model.Player{ID: "player-2"}, Role: game.GameRoleHunter},
		game.Player{Player: model.Player{ID: "player-3"}, Role: game.GameRoleTarget},
	}
	expectedGame := game.NewGameWithParams(gameID, true, players, "player-3")

	gameEvt := GameEvent{Game: expectedGame, Event: game.GameEventNothing}
	serialized, _ := json.Marshal(gameEvt)
	repo.On("FeatureExtraData", "game", gameID).Return(string(serialized), nil)

	game, evt, err := service.GameByID(gameID)

	assert.NoError(t, err)
	assert.Equal(t, expectedGame, game)
	assert.Equal(t, gameEvt.Event, *evt)
}

func TestGameServiceMustObserveGameChangeEvents(t *testing.T) {
	repo := &repo_mocks.Repository{}
	stream := &repo_mocks.EventStream{}
	dispatcher := &messages_mocks.Dispatcher{}
	service := NewGameService(repo, stream, dispatcher)

	g, e := game.NewGame(gameID)

	dispatcher.On("Subscribe", GameChangeTopic, mock.MatchedBy(func(fn func(data []byte) error) bool {
		gameEvt := GameEvent{Game: g, Event: e, LastUpdate: time.Now(), ServerID: serverID}
		data, _ := json.Marshal(gameEvt)
		err := fn(data)
		return assert.NoError(t, err)
	})).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := service.ObserveGamesEvents(ctx, func(actualG *game.Game, actualE game.Event) error {
		assert.Equal(t, g, actualG)
		assert.Equal(t, e, actualE)
		return nil
	})
	require.NoError(t, err)
}

func TestGameServiceGamesAroundPlayer(t *testing.T) {
	repo := &repo_mocks.Repository{}
	service := NewGameService(repo, nil, nil)

	expected := []GameWithCoords{
		GameWithCoords{Game: &game.Game{ID: "game-test-1"}, Coords: "fake-coords-1"},
		GameWithCoords{Game: &game.Game{ID: "game-test-2"}, Coords: "fake-coords-2"},
	}

	repo.On("FeaturesAround", mock.Anything, mock.Anything).Return([]*model.Feature{
		&model.Feature{ID: expected[0].ID, Coordinates: expected[0].Coords},
		&model.Feature{ID: expected[1].ID, Coordinates: expected[1].Coords},
	}, nil)

	player := model.Player{ID: "player-test-1", Lat: 0, Lon: 0}
	gamesAround, err := service.GamesAround(player)
	require.NoError(t, err)
	assert.EqualValues(t, expected, gamesAround)
}

func assertDateEqual(t *testing.T, date1, date2 time.Time) bool {
	return assert.Condition(t, func() bool {
		return assert.Equal(t, date1.Day(), date2.Day()) &&
			assert.Equal(t, date1.Month(), date2.Month()) &&
			assert.Equal(t, date1.Year(), date2.Year())
	})
}

func matchGameChangePayload(t *testing.T) interface{} {
	now := time.Now()
	return mock.MatchedBy(func(data interface{}) bool {
		var payload string
		switch data.(type) {
		case string:
			payload = data.(string)
		case []byte:
			payload = string(data.([]byte))
		}
		gameEvt := GameEvent{}
		json.Unmarshal([]byte(payload), &gameEvt)

		return assert.Equal(t, gameEvt.Event, game.GameEventCreated) &&
			assert.Equal(t, serverID, gameEvt.ServerID) &&
			assertDateEqual(t, now, gameEvt.LastUpdate)
	})
}
