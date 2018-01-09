// Code generated by mockery v1.0.0
package mocks

import context "context"
import game "github.com/perenecabuto/CatchCatch/catchcatch-server/game"
import mock "github.com/stretchr/testify/mock"
import model "github.com/perenecabuto/CatchCatch/catchcatch-server/model"

// GameService is an autogenerated mock type for the GameService type
type GameService struct {
	mock.Mock
}

// Create provides a mock function with given fields: gameID, serverID
func (_m *GameService) Create(gameID string, serverID string) error {
	ret := _m.Called(gameID, serverID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(gameID, serverID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GameByID provides a mock function with given fields: gameID
func (_m *GameService) GameByID(gameID string) (*game.Game, *game.GameEvent, error) {
	ret := _m.Called(gameID)

	var r0 *game.Game
	if rf, ok := ret.Get(0).(func(string) *game.Game); ok {
		r0 = rf(gameID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*game.Game)
		}
	}

	var r1 *game.GameEvent
	if rf, ok := ret.Get(1).(func(string) *game.GameEvent); ok {
		r1 = rf(gameID)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*game.GameEvent)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string) error); ok {
		r2 = rf(gameID)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// IsGameRunning provides a mock function with given fields: gameID
func (_m *GameService) IsGameRunning(gameID string) (bool, error) {
	ret := _m.Called(gameID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(gameID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(gameID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ObserveGamePlayers provides a mock function with given fields: ctx, gameID, callback
func (_m *GameService) ObserveGamePlayers(ctx context.Context, gameID string, callback func(model.Player, bool) error) error {
	ret := _m.Called(ctx, gameID, callback)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, func(model.Player, bool) error) error); ok {
		r0 = rf(ctx, gameID, callback)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ObserveGamesEvents provides a mock function with given fields: ctx, callback
func (_m *GameService) ObserveGamesEvents(ctx context.Context, callback func(*game.Game, *game.GameEvent) error) error {
	ret := _m.Called(ctx, callback)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, func(*game.Game, *game.GameEvent) error) error); ok {
		r0 = rf(ctx, callback)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ObservePlayersCrossGeofences provides a mock function with given fields: ctx, callback
func (_m *GameService) ObservePlayersCrossGeofences(ctx context.Context, callback func(string, model.Player) error) error {
	ret := _m.Called(ctx, callback)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, func(string, model.Player) error) error); ok {
		r0 = rf(ctx, callback)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Remove provides a mock function with given fields: gameID
func (_m *GameService) Remove(gameID string) error {
	ret := _m.Called(gameID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(gameID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Update provides a mock function with given fields: g, serverID, evt
func (_m *GameService) Update(g *game.Game, serverID string, evt game.GameEvent) error {
	ret := _m.Called(g, serverID, evt)

	var r0 error
	if rf, ok := ret.Get(0).(func(*game.Game, string, game.GameEvent) error); ok {
		r0 = rf(g, serverID, evt)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}