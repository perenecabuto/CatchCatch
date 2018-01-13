package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/mocks"
	"github.com/perenecabuto/CatchCatch/server/model"
)

func TestNewGameWorker(t *testing.T) {
	t.Parallel()

	serverID := "test-gameworker-server-1"
	gameID := "test-gameworker-game-1"
	playerIDs := []string{
		"test-gameworker-player-1",
		"test-gameworker-player-2",
		"test-gameworker-player-3",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gameService, wait := newMockedGameService(ctx,
		serverID, gameID, playerIDs)
	defer close(wait)
	w := NewGameWorker(serverID, gameService)

	err := w.WatchGames(ctx)
	assert.NoError(t, err)
	<-wait

	gameService.AssertCalled(t, "IsGameRunning", gameID)

	matchGameID := mock.MatchedBy(func(g *game.Game) bool {
		return assert.Equal(t, gameID, g.ID)
	})
	matchGameEvent := mock.MatchedBy(func(evt game.GameEvent) bool {
		expected := game.GameStarted
		return assert.Equal(t, expected, evt.Name)
	})
	gameService.AssertCalled(t, "Update", matchGameID, serverID, matchGameEvent)
}

func TestCloseWhenFinish(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	serverID := "test-gameworker-server-1"
	gameID := "test-gameworker-game-1"
	playerIDs := []string{
		"test-gameworker-player-1",
		"test-gameworker-player-2",
		"test-gameworker-player-3",
	}

	gameService, _ := newMockedGameService(ctx, serverID, gameID, playerIDs)
	w := NewGameWorker(serverID, gameService)
	w.WatchGames(ctx)
	<-time.NewTimer(time.Second).C
	cancel()
	<-time.NewTimer(time.Second).C

	matchGameID := mock.MatchedBy(func(g *game.Game) bool {
		return assert.Equal(t, gameID, g.ID)
	})
	gameService.AssertCalled(t, "Update", matchGameID, serverID, mock.AnythingOfType("game.GameEvent"))
	gameService.AssertCalled(t, "Remove", gameID)
}

func newMockedGameService(ctx context.Context, serverID, gameID string, playerIDs []string) (*mocks.GameService, chan interface{}) {
	gameService := new(mocks.GameService)
	gameService.On("Remove", gameID).Return(nil)

	gameService.On("ObservePlayersCrossGeofences",
		ctx, mock.MatchedBy(func(fn func(string, model.Player) error) bool {
			fn(gameID, model.Player{})
			return true
		}),
	).Return(nil)

	gameService.On("IsGameRunning", gameID).Return(false, nil)
	gameService.On("Create", gameID, serverID).Return(nil)
	gameService.On("Remove", gameID).Return(nil)
	gameService.On("Update", gameID, mock.Anything, mock.Anything).Return(nil)

	wait := make(chan interface{})

	gameService.On("ObservePlayersCrossGeofences",
		ctx, mock.MatchedBy(func(fn func(string, model.Player) error) bool {
			fn(gameID, model.Player{})
			wait <- new(interface{})
			return true
		}),
	).Return(nil)

	gameService.On("ObserveGamePlayers", mock.Anything, gameID,
		mock.MatchedBy(func(fn func(model.Player, bool) error) bool {
			for _, id := range playerIDs {
				p, exit := model.Player{ID: id, Lon: 0, Lat: 0}, false
				fn(p, exit)
			}
			go func() {
				<-time.NewTimer(time.Second).C
				wait <- new(interface{})
			}()
			return true
		}),
	).Return(nil)

	gameService.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return gameService, wait
}
