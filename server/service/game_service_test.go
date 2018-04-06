package service_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service"

	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
)

const (
	gameID = "test-game-service-game1"
)

func TestGameServiceCreate(t *testing.T) {
	repo := &smocks.Repository{}
	stream := &smocks.EventStream{}
	gService := service.NewGameService(repo, stream)

	f := &model.Feature{ID: "new-game-test-1", Coordinates: "fake-coords-1"}
	repo.On("SetFeature", "game", f.ID, f.Coordinates).
		Return(f, nil)
	repo.On("SetFeatureExtraData", "game", f.ID, mock.Anything).
		Return(nil)

	_, err := gService.Create(f.ID, f.Coordinates)
	require.NoError(t, err)

	example := game.NewGame(f.ID)

	repo.AssertCalled(t, "SetFeature", "game", f.ID, f.Coordinates)
	repo.AssertCalled(t, "SetFeatureExtraData", "game", f.ID,
		mock.MatchedBy(func(data string) bool {
			actual := &game.Game{}
			json.Unmarshal([]byte(data), actual)
			return assert.Equal(t, example, actual)
		}))
}

func TestGameServiceUpdate(t *testing.T) {
	repo := &smocks.Repository{}
	stream := &smocks.EventStream{}
	gService := service.NewGameService(repo, stream)

	repo.On("SetFeatureExtraData", "game", gameID, mock.Anything).
		Return(nil)

	g := &service.GameWithCoords{Game: game.NewGame(gameID)}

	err := gService.Update(g)
	require.NoError(t, err)

	example := game.NewGame(gameID)

	repo.AssertCalled(t, "SetFeatureExtraData", "game", gameID,
		mock.MatchedBy(func(data string) bool {
			actual := &game.Game{}
			json.Unmarshal([]byte(data), actual)
			return assert.Equal(t, example, actual)
		}))
}

func TestGameServiceMustRetrieveUpdatedGame(t *testing.T) {
	repo := &smocks.Repository{}
	stream := &smocks.EventStream{}
	gService := service.NewGameService(repo, stream)

	players := make([]game.Player, 0)

	repo.On("SetFeatureExtraData", "game", gameID, mock.Anything).Return(nil)

	g := game.NewGameWithParams(gameID, false, players, "")
	example := &service.GameWithCoords{Game: g}
	gService.Update(example)

	f := &model.Feature{Group: "game", ID: g.ID}
	repo.On("FeatureByID", "game", gameID).Return(f, nil)
	data, _ := json.Marshal(g)
	repo.On("FeatureExtraData", "game", gameID).Return(string(data), nil)

	actual, err := gService.GameByID(gameID)
	assert.NoError(t, err)
	assert.Equal(t, example, actual)
}

func TestGameServiceMustGetGameWithPlayers(t *testing.T) {
	repo := &smocks.Repository{}
	stream := &smocks.EventStream{}
	gService := service.NewGameService(repo, stream)

	players := []game.Player{
		game.Player{Player: model.Player{ID: "player-1"}, Role: game.GameRoleHunter},
		game.Player{Player: model.Player{ID: "player-2"}, Role: game.GameRoleHunter},
		game.Player{Player: model.Player{ID: "player-3"}, Role: game.GameRoleTarget},
	}

	g := game.NewGameWithParams(gameID, true, players, "player-3")
	example := &service.GameWithCoords{Game: g}
	f := &model.Feature{Group: "game", ID: g.ID}
	repo.On("FeatureByID", "game", gameID).Return(f, nil)
	data, _ := json.Marshal(g)
	repo.On("FeatureExtraData", "game", gameID).Return(string(data), nil)

	actual, err := gService.GameByID(gameID)
	assert.NoError(t, err)
	assert.Equal(t, example, actual)
}

func TestGameServiceGamesAroundPlayer(t *testing.T) {
	repo := &smocks.Repository{}
	gService := service.NewGameService(repo, nil)

	expected := []service.GameWithCoords{
		service.GameWithCoords{Game: game.NewGame("game-test-1"), Coords: "fake-coords-1"},
		service.GameWithCoords{Game: game.NewGame("game-test-2"), Coords: "fake-coords-2"},
	}

	repo.On("FeaturesAround", mock.Anything, mock.Anything).Return([]*model.Feature{
		&model.Feature{ID: expected[0].ID, Coordinates: expected[0].Coords},
		&model.Feature{ID: expected[1].ID, Coordinates: expected[1].Coords},
	}, nil)

	for _, e := range expected {
		g := game.NewGameWithParams(e.ID, false, []game.Player{}, "")
		data, _ := json.Marshal(g)
		repo.On("FeatureExtraData", "game", g.ID).Return(string(data), nil)
	}

	player := model.Player{ID: "player-test-1", Lat: 0, Lon: 0}
	actual, err := gService.GamesAround(player)
	require.NoError(t, err)
	jsonE, _ := json.Marshal(expected)
	jsonA, _ := json.Marshal(actual)
	assert.JSONEq(t, string(jsonE), string(jsonA))
}

func assertDateEqual(t *testing.T, date1, date2 time.Time) bool {
	return assert.Condition(t, func() bool {
		return assert.Equal(t, date1.Day(), date2.Day()) &&
			assert.Equal(t, date1.Month(), date2.Month()) &&
			assert.Equal(t, date1.Year(), date2.Year())
	})
}
