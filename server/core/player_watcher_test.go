package core_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/perenecabuto/CatchCatch/server/service/messages"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service/mocks"
)

var any = mock.Anything

func TestPlayerWatcherTaskID(t *testing.T) {
	w := core.NewPlayersWatcher(nil, nil)
	assert.Equal(t, "PlayersWatcher", w.ID())
}

func TestPlayerWatcherWhenRunObservePlayerDeleteEventAndPublishIt(t *testing.T) {
	p := &mocks.PlayerLocationService{}
	m := &mocks.Dispatcher{}
	w := core.NewPlayersWatcher(m, p)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expected := model.Player{ID: "deleted-player-1"}
	p.On("ObservePlayerDelete", any, any).
		Run(func(args mock.Arguments) {
			cb := args[1].(func(p model.Player) error)
			err := cb(expected)
			assert.NoError(t, err)
		}).Return(nil)

	resultChan := make(chan []byte)
	m.On("Publish", any, mock.MatchedBy(func(data []byte) bool {
		resultChan <- data
		return true
	})).Return(nil)

	go w.Run(ctx, nil)

	received := <-resultChan
	p.AssertCalled(t, "ObservePlayerDelete", any, any)
	m.AssertCalled(t, "Publish", core.PlayerWatcherEventDel, received)
}

func TestPlayerWatcherOnPlayerDeletedSubscribesAndReceiveDeletedPlayer(t *testing.T) {
	p := &mocks.PlayerLocationService{}
	m := &mocks.Dispatcher{}
	w := core.NewPlayersWatcher(m, p)

	expected := &model.Player{ID: "deleted-player-1", Lat: 10, Lon: 11}
	m.On("Subscribe", any, any, any).Run(func(args mock.Arguments) {
		cb := args[2].(messages.OnMessage)
		data, _ := json.Marshal(expected)
		err := cb(data)
		assert.NoError(t, err)
	}).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultChan := make(chan *model.Player)
	go w.OnPlayerDeleted(ctx, func(p *model.Player) error {
		resultChan <- p
		return nil
	})

	received := <-resultChan

	m.AssertCalled(t, "Subscribe", any, core.PlayerWatcherEventDel, any)
	assert.EqualValues(t, expected, received)
}

func TestPlayerWatcherOnPlayerDeletedSubscribesDoNotReceiveDeletedPlayerWhenDataIsCorrupted(t *testing.T) {
	p := &mocks.PlayerLocationService{}
	m := &mocks.Dispatcher{}
	w := core.NewPlayersWatcher(m, p)

	resultChan := make(chan error)
	m.On("Subscribe", any, any, any).Run(func(args mock.Arguments) {
		cb := args[2].(messages.OnMessage)
		go func() {
			resultChan <- cb([]byte("a234advzz!!!"))
		}()
	}).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var callbackExecuted = false
	w.OnPlayerDeleted(ctx, func(p *model.Player) error {
		callbackExecuted = true
		return nil
	})

	err := <-resultChan
	assert.Error(t, err)
	assert.False(t, callbackExecuted)
}
