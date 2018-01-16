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

func assertDateEqual(t *testing.T, date1, date2 time.Time) bool {
	return assert.Condition(t, func() bool {
		return assert.Equal(t, date1.Day(), date2.Day()) &&
			assert.Equal(t, date1.Month(), date2.Month()) &&
			assert.Equal(t, date1.Year(), date2.Year())
	})
}
