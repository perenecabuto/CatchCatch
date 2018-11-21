package core

import (
	"context"
	"encoding/json"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/service/messages"
	"github.com/perenecabuto/CatchCatch/server/worker"
	"github.com/pkg/errors"
)

// PlayerWatcherEvents
const (
	PlayerWatcherEventDel = "players:delete"
)

// PlayersWatcher run background job listening to players events
// and notify it to subscribers
type PlayersWatcher struct {
	players  service.PlayerLocationService
	messages messages.Dispatcher
}

var _ worker.Task = (*PlayersWatcher)(nil)

// NewPlayersWatcher creates a PlayersWatcher
func NewPlayersWatcher(m messages.Dispatcher, p service.PlayerLocationService) *PlayersWatcher {
	return &PlayersWatcher{players: p, messages: m}
}

// ID return the PlayerWatcher task ID
func (w *PlayersWatcher) ID() string {
	return "PlayersWatcher"
}

// OnPlayerDeleted subscribes to player delete channel
// and respond with deleted player object
func (w *PlayersWatcher) OnPlayerDeleted(ctx context.Context, cb func(*model.Player) error) error {
	return w.messages.Subscribe(ctx, PlayerWatcherEventDel, func(data []byte) error {
		payload := &model.Player{}
		err := json.Unmarshal(data, payload)
		if err != nil {
			return errors.Cause(err)
		}
		return cb(payload)
	})
}

/*
Run is a worker.Task Run implementation
it observes player deleted and publishes the event
*/
func (w *PlayersWatcher) Run(ctx context.Context, params worker.TaskParams) error {
	return w.players.ObservePlayerDelete(ctx, func(p model.Player) error {
		data, _ := json.Marshal(&p)
		err := w.messages.Publish(PlayerWatcherEventDel, data)
		return errors.Cause(err)
	})
}
