package game

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	players = []Player{
		{Player: model.Player{ID: "hunter-1"}, Lose: false, Role: GameRoleHunter},
		{Player: model.Player{ID: "hunter-2"}, Lose: true, Role: GameRoleHunter, DistToTarget: 10},
		{Player: model.Player{ID: "hunter-3", Lon: 42, Lat: 31}, Lose: false, Role: GameRoleHunter},
		{Player: model.Player{ID: "target-1"}, Lose: false, Role: GameRoleTarget},
	}
	exampleGame = NewGameWithParams("game-test-1", true, players, "target-1")

	exampleGameJSONString = strings.TrimSpace(`
		{
			"id": "game-test-1",
			"started": 1,
			"targetID": "target-1",
			"players": {
				"hunter-1": {"id":"hunter-1", "Lose":false, "lon":0, "lat":0, "Role":"hunter", "DistToTarget":0},
				"hunter-2": {"id":"hunter-2", "Lose":true , "lon":0, "lat":0, "Role":"hunter", "DistToTarget":10},
				"hunter-3": {"id":"hunter-3", "Lose":false, "lon":42, "lat":31, "Role":"hunter", "DistToTarget":0},
				"target-1": {"id":"target-1", "Lose":false, "lon":0, "lat":0, "Role":"target", "DistToTarget":0}
			}
		}
	`)
)

func TestGameMarshaler(t *testing.T) {
	serialized, err := json.Marshal(exampleGame)
	require.NoError(t, err)
	assert.JSONEq(t, exampleGameJSONString, string(serialized))
}

func TestGameUnmarshaler(t *testing.T) {
	deserialized := &Game{}
	err := json.Unmarshal([]byte(exampleGameJSONString), deserialized)
	require.NoError(t, err)
	assert.Equal(t, exampleGame, deserialized)
}

func TestGameUnmarshalerReturnErrorWhenPlayerListIsInvalid(t *testing.T) {
	example := strings.TrimSpace(`{"id": "game-test-1", "players": {"ovo": ""}}`)
	deserialized := &Game{}
	err := json.Unmarshal([]byte(example), deserialized)
	assert.Error(t, err)
}
