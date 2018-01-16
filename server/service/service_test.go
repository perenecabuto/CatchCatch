package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gjson "github.com/tidwall/gjson"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/mocks/repo_mocks"
	"github.com/perenecabuto/CatchCatch/server/model"
)

var (
	gameID   = "test-tile38service-game1"
	serverID = "test-tile38service-server1"
)

func TestTile38LocationServiceCreate(t *testing.T) {
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
		actualServerID := gjson.Get(payload, "server_id").String()
		actualUpdatedAt := gjson.Get(payload, "updated_at").Int()
		updateDate := time.Unix(actualUpdatedAt, 0)

		return assert.Equal(t, actualServerID, serverID) &&
			assertDateEqual(t, now, updateDate)
	})
	repo.AssertCalled(t, "SetFeatureExtraData", "game",
		gameID, matchPayload)
}

func TestTile38LocationServiceMustGetNewGame(t *testing.T) {
	repo := &repo_mocks.Repository{}
	stream := &repo_mocks.EventStream{}
	service := NewGameService(repo, stream)

	repo.On("FeatureExtraData", "game", gameID).
		Return("", nil)

	players := map[string]*game.Player{}
	expectedGame := game.NewGameWithParams(gameID, false, players, "")

	game, evt, err := service.GameByID(gameID)
	assert.NoError(t, err)
	assert.Equal(t, expectedGame, game)
	assert.NotNil(t, evt)
}

func TestTile38LocationServiceMustGetGameWithPlayers(t *testing.T) {
	repo := &repo_mocks.Repository{}
	stream := &repo_mocks.EventStream{}
	service := NewGameService(repo, stream)

	repo.On("FeatureExtraData", "game", gameID).
		Return(`{
			"started": true,
			"players": [
			{"id": "player-1", "Role": "hunter", "DistToTarget": 0, "Loose": false},
			{"id": "player-2", "Role": "hunter", "DistToTarget": 0, "Loose": false},
			{"id": "player-3", "Role": "target", "DistToTarget": 0, "Loose": false},
		]}`, nil)

	players := map[string]*game.Player{
		"player-1": &game.Player{Player: model.Player{ID: "player-1"}, Role: game.GameRoleHunter},
		"player-2": &game.Player{Player: model.Player{ID: "player-2"}, Role: game.GameRoleHunter},
		"player-3": &game.Player{Player: model.Player{ID: "player-3"}, Role: game.GameRoleTarget},
	}
	expectedGame := game.NewGameWithParams(gameID, true, players, "player-3")

	game, evt, err := service.GameByID(gameID)
	assert.NoError(t, err)
	assert.Equal(t, expectedGame, game)
	assert.NotNil(t, evt)
}

func assertDateEqual(t *testing.T, date1, date2 time.Time) bool {
	return assert.Condition(t, func() bool {
		return assert.Equal(t, date1.Day(), date2.Day()) &&
			assert.Equal(t, date1.Month(), date2.Month()) &&
			assert.Equal(t, date1.Year(), date2.Year())
	})
}
