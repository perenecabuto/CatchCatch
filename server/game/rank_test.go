package game_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
)

func TestGameRank(t *testing.T) {
	players := []game.Player{
		{Player: model.Player{ID: "player1", Lat: 0, Lon: 0}, Role: game.GameRoleHunter},
		{Player: model.Player{ID: "player2", Lat: 0, Lon: 0}, Role: game.GameRoleHunter},
		{Player: model.Player{ID: "target", Lat: 0, Lon: 0}, Role: game.GameRoleTarget},
	}

	g := game.NewGameWithParams("test", true, players, "target")

	g.SetPlayer("target", 0.0, 0.0)
	g.SetPlayer("player2", 0.0, 0.1)
	g.SetPlayer("player1", 0.1, 0.1)

	rank := g.Rank()
	rankByPlayer := make(map[string]int)
	for _, r := range rank.PlayerRank {
		rankByPlayer[r.Player] = r.Points
	}

	expectedRank := map[string]int{
		"player1": 99,
		"player2": 70,
		"target":  0,
	}
	for pID, points := range expectedRank {
		assert.Equal(t, points, rankByPlayer[pID])
	}
}
