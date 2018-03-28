package core

import (
	"context"
	"testing"

	"github.com/perenecabuto/CatchCatch/server/mocks"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/websocket"
)

func TestNewAdminWatcher(t *testing.T) {
	if w := createAdminWatcher(); w == nil {
		t.Fatal("Can't create AdminWatcher")
	}
}

func createAdminWatcher() *AdminWatcher {
	wss := websocket.NewWSServer(&mocks.WSDriver{})
	playerService := &MockPlayerServiceWithCallback{}
	return NewAdminWatcher(playerService, wss)
}

type MockPlayerServiceWithCallback struct {
	mocks.PlayerLocationService
	PlayersAroundCallback       service.PlayersAroundCallback
	PlayerNearToFeatureCallback service.PlayerNearToFeatureCallback
}

func (gs *MockPlayerServiceWithCallback) ObservePlayersAround(_ context.Context, cb service.PlayersAroundCallback) error {
	gs.PlayersAroundCallback = cb
	return nil
}
func (gs *MockPlayerServiceWithCallback) ObservePlayerNearToFeature(_ context.Context, _ string, cb service.PlayerNearToFeatureCallback) error {
	gs.PlayerNearToFeatureCallback = cb
	return nil
}
