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

	gameID := "test-gameworker-game-1"
	playerIDs := []string{
		"test-gameworker-player-1",
		"test-gameworker-player-2",
		"test-gameworker-player-3",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gameService, wait := newMockedGameService(ctx, gameID, playerIDs)
	defer close(wait)
	w := NewGameWorker(gameService)

	err := w.Run(ctx, nil)
	assert.NoError(t, err)
	<-wait

	matchGameID := mock.MatchedBy(func(g *game.Game) bool {
		return assert.Equal(t, gameID, g.ID)
	})
	matchedEvent := mock.MatchedBy(func(evt game.Event) bool {
		expected := game.GameStarted
		return assert.Equal(t, expected, evt.Name)
	})
	gameService.AssertCalled(t, "Update", matchGameID, matchedEvent)
}

func TestCloseWhenFinish(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	gameID := "test-gameworker-game-1"
	playerIDs := []string{
		"test-gameworker-player-1",
		"test-gameworker-player-2",
		"test-gameworker-player-3",
	}

	gameService, _ := newMockedGameService(ctx, gameID, playerIDs)
	w := NewGameWorker(gameService)
	w.Run(ctx, nil)
	<-time.NewTimer(time.Second).C
	cancel()
	<-time.NewTimer(time.Second).C

	matchGameID := mock.MatchedBy(func(g *game.Game) bool {
		return assert.Equal(t, gameID, g.ID)
	})
	gameService.AssertCalled(t, "Update", matchGameID, mock.AnythingOfType("game.Event"))
	gameService.AssertCalled(t, "Remove", gameID)
}

func newMockedGameService(ctx context.Context, gameID string, playerIDs []string) (*mocks.GameService, chan interface{}) {
	gameService := new(mocks.GameService)
	gameService.On("Remove", gameID).Return(nil)

	gameService.On("ObservePlayersCrossGeofences",
		ctx, mock.MatchedBy(func(fn func(string, model.Player) error) bool {
			go fn(gameID, model.Player{})
			return true
		}),
	).Return(nil)

	g, _ := game.NewGame(gameID)
	gameService.On("Create", gameID).Return(g, nil)
	gameService.On("Remove", gameID).Return(nil)
	gameService.On("Update", gameID, mock.Anything, mock.Anything).Return(nil)

	wait := make(chan interface{})

	gameService.On("ObservePlayersCrossGeofences",
		ctx, mock.MatchedBy(func(fn func(string, model.Player) error) bool {
			go fn(gameID, model.Player{})
			go func() { wait <- new(interface{}) }()
			return true
		}),
	).Return(nil)

	gameService.On("ObserveGamePlayers", mock.Anything, gameID,
		mock.MatchedBy(func(fn func(model.Player, bool) error) bool {
			go func() {
				<-time.NewTimer(time.Second).C
				wait <- new(interface{})
			}()
			go func() {
				for _, id := range playerIDs {
					p, exit := model.Player{ID: id, Lon: 0, Lat: 0}, false
					go fn(p, exit)
				}
			}()
			return true
		}),
	).Return(nil)

	gameService.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return gameService, wait
}
