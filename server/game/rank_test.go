package game_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
)

func TestGameRankString(t *testing.T) {
	r := game.NewGameRank("test-game")
	players := []game.Player{
		{Player: model.Player{ID: "player1"}, Role: game.GameRoleHunter, DistToTarget: 10, Lose: false},
		{Player: model.Player{ID: "player2"}, Role: game.GameRoleHunter, DistToTarget: 10, Lose: true},
		{Player: model.Player{ID: "target"}, Role: game.GameRoleTarget, DistToTarget: 0, Lose: false},
	}

	actual := r.ByPlayersDistanceToTarget(players).String()
	expected := "[Game:test-game Rank:[\nplayer1 = 9 \nplayer2 = 0 \ntarget = 100]]"

	assert.Equal(t, expected, actual)
}

func TestGameRankWhenTargetLoses(t *testing.T) {
	players := []game.Player{
		{Player: model.Player{ID: "player1"}, DistToTarget: 5, Role: game.GameRoleHunter, Lose: false},
		{Player: model.Player{ID: "player2"}, DistToTarget: 10, Role: game.GameRoleHunter, Lose: false},
		{Player: model.Player{ID: "target"}, DistToTarget: 0, Role: game.GameRoleTarget, Lose: true},
	}

	rank := game.NewGameRank("test").ByPlayersDistanceToTarget(players)
	rankByPlayer := make(map[string]int)
	for _, r := range rank.PlayerRank {
		rankByPlayer[r.Player.ID] = r.Points
	}

	expectedRank := map[string]int{
		"player1": 55,
		"player2": 9,
		"target":  0,
	}

	assert.EqualValues(t, expectedRank, rankByPlayer)
}

func TestGameRankWhenTargetWins(t *testing.T) {
	players := []game.Player{
		{Player: model.Player{ID: "player1"}, DistToTarget: 5, Role: game.GameRoleHunter, Lose: false},
		{Player: model.Player{ID: "player2"}, DistToTarget: 90, Role: game.GameRoleHunter, Lose: false},
		{Player: model.Player{ID: "target"}, DistToTarget: 0, Role: game.GameRoleTarget, Lose: false},
	}

	rank := game.NewGameRank("test").ByPlayersDistanceToTarget(players)
	rankByPlayer := make(map[string]int)
	for _, r := range rank.PlayerRank {
		rankByPlayer[r.Player.ID] = r.Points
	}

	expectedRank := map[string]int{
		"player1": 95,
		"player2": 1,
		"target":  100,
	}

	assert.EqualValues(t, expectedRank, rankByPlayer)
}

func TestGameRankWhenPlayerListIsEmpty(t *testing.T) {
	expected := *game.NewGameRank("test")
	actual := game.NewGameRank("test").ByPlayersDistanceToTarget([]game.Player{})

	assert.EqualValues(t, expected, actual)
}
