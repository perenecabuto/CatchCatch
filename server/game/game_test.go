package game

import (
	"strconv"
	"testing"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGameStringFormat(t *testing.T) {
	g := NewGameWithParams("test", true,
		[]Player{Player{model.Player{ID: "1", Lat: 0, Lon: 1}, GameRoleHunter, 1, false}}, "target")
	actual := g.String()
	expected := "[ID: test, Started: true, Players: [[ID: 1, Role: hunter, DistToTarget: 1.000000, Lose: false]]]"

	assert.Equal(t, expected, actual)
}

func TestGameMustAddPlayers(t *testing.T) {
	g := NewGame("test")
	g.SetPlayer("player1", 0, 0)
	g.SetPlayer("player2", 0, 0)
	g.SetPlayer("player3", 0, 0)

	assert.Len(t, g.Players(), 3)
}

func TestGameDoNotAddPlayersWhenItIsStarted(t *testing.T) {
	g := NewGame("test")
	g.SetPlayer("player1", 0, 0)
	g.SetPlayer("player2", 0, 0)
	g.SetPlayer("player3", 0, 0)

	g.Start()
	assert.Len(t, g.Players(), 3)

	evt := g.SetPlayer("player4", 0, 0)
	assert.Equal(t, evt, GameEventNothing)
	assert.Len(t, g.Players(), 3)
}

func TestGameTargetIDIsEmptyWhenItStartsWithoutPlayers(t *testing.T) {
	g := NewGame("test")
	g.Start()

	assert.Equal(t, g.TargetID(), "")
}

func TestGameReturnPlayerNearToTargetWhenHunterIsCloser(t *testing.T) {
	hunterID := "hunter-1"
	players := []Player{
		{Player: model.Player{ID: hunterID, Lat: 1, Lon: 1}, Role: GameRoleHunter},
		{Player: model.Player{ID: "target", Lat: 0, Lon: 0}, Role: GameRoleTarget},
	}
	g := NewGameWithParams("game", true, players, "target")

	expected := Event{
		Name: GamePlayerNearToTarget,
		Player: Player{
			Player:       model.Player{ID: hunterID, Lat: 0.0002, Lon: 0.0002},
			DistToTarget: 31.45067466553135,
			Role:         GameRoleHunter,
			Lose:         false,
		},
	}
	evt := g.SetPlayer(hunterID, expected.Player.Lon, expected.Player.Lat)

	assert.Equal(t, expected, evt)
}

func TestGameMustSetPlayersRolesOnStart(t *testing.T) {
	g := NewGame("test")
	totalPlayers := 6
	for i := 1; i <= totalPlayers; i++ {
		g.SetPlayer("player"+strconv.Itoa(i), 0, 0)
	}
	for _, p := range g.Players() {
		require.Equal(t, GameRoleUndefined, p.Role)
	}

	g.Start()
	for _, p := range g.Players() {
		require.NotEqual(t, GameRoleUndefined, p.Role)
	}
	roles := make(map[Role]int)
	for _, p := range g.Players() {
		roles[p.Role]++
	}

	assert.Equal(t, totalPlayers-1, roles[GameRoleHunter])
	assert.Equal(t, 1, roles[GameRoleTarget])
}

func TestGameMustSetDistToTargetWhenStart(t *testing.T) {
	g := NewGame("test")
	g.SetPlayer("player1", 0.0, 0.0)
	g.SetPlayer("player2", 0.0, 0.0)
	g.SetPlayer("target", 0.0, 0.0)
	g.Start()

	g.players["player1"].Role = GameRoleHunter
	g.players["player2"].Role = GameRoleHunter
	g.players["target"].Role = GameRoleTarget
	g.targetID = "target"

	g.SetPlayer("player1", 0.1, 0.01)
	g.SetPlayer("player2", 0.01, 0.01)
	g.SetPlayer("target", 0.001, 0.001)

	expectedDists := map[string]float64{
		"player1": 11174.951768601733,
		"player2": 1572.5337292863205,
		"target":  0,
	}
	for _, p := range g.Players() {
		assert.Equal(t, expectedDists[p.ID], p.DistToTarget)
	}
}

func TestGamePlayersDistToTargetMustBeConsistent(t *testing.T) {
	g := NewGame("test")
	g.SetPlayer("player1", 0, 0)
	g.SetPlayer("player2", 0.00001, 0)
	g.SetPlayer("player3", 0.0001, 0.00001)
	g.Start()

	playersAfterStart := make([]Player, len(g.Players()))
	copy(playersAfterStart, g.Players())

	g.SetPlayer("player1", 0, 0)
	g.SetPlayer("player2", 0.00001, 0)
	g.SetPlayer("player3", 0.0001, 0.00001)

	playersAfterSet := make(map[string]Player)
	for _, p := range g.Players() {
		playersAfterSet[p.ID] = p
	}
	for _, p := range playersAfterStart {
		pAfterSet := playersAfterSet[p.ID]
		assert.Equal(t, p.DistToTarget, pAfterSet.DistToTarget)
	}
}

func TestGameRank(t *testing.T) {
	g := NewGame("test")
	g.SetPlayer("player1", 0.0, 0.0)
	g.SetPlayer("player2", 0.0, 0.0)
	g.SetPlayer("target", 0.0, 0.0)
	g.Start()

	g.players["player1"].Role = GameRoleHunter
	g.players["player2"].Role = GameRoleHunter
	g.players["target"].Role = GameRoleTarget
	g.targetID = "target"

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

func TestGameReturnsTheTargetPlayerWhenItIsSet(t *testing.T) {
	targetID := "test-game-1-target"
	targetPlayer := Player{Player: model.Player{ID: targetID}}
	otherPlayer := Player{Player: model.Player{ID: "1234"}}
	g := NewGameWithParams("test-game-1", false, []Player{
		targetPlayer, otherPlayer,
	}, targetID)

	assert.Equal(t, g.TargetPlayer(), &targetPlayer)
}

func TestGameReturnsNilWhenTargetPlayerIsNotSet(t *testing.T) {
	targetID := "test-game-1-target"
	otherPlayer := Player{Player: model.Player{ID: "1234"}}
	g := NewGameWithParams("test-game-1", false, []Player{
		otherPlayer, otherPlayer,
	}, targetID)

	assert.Nil(t, g.TargetPlayer())
}

func TestGameClenUpWhenStop(t *testing.T) {
	targetID := "test-game-1-target"
	targetPlayer := Player{Player: model.Player{ID: targetID}}
	otherPlayer := Player{Player: model.Player{ID: "1234"}}
	g := NewGameWithParams("test-game-1", true, []Player{
		targetPlayer, otherPlayer, otherPlayer,
	}, targetID)

	g.Stop()

	assert.Empty(t, g.TargetID())
	assert.False(t, g.Started())
	assert.Empty(t, g.Players())
}
