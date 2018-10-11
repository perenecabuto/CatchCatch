package core_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	core "github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/model"
	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
)

func TestFeaturesWatcherNotifiesFeaturesEventsToAdmin(t *testing.T) {
	m := &smocks.Dispatcher{}
	p := &smocks.PlayerLocationService{}
	w := core.NewFeaturesEventsWatcher(m, p)

	example := &core.FeatureEventsNearToAdminPayload{
		AdminID: "test-admin-id",
		Feature: model.Feature{ID: "test-geofence-id", Group: "geofence", Coordinates: "[-1, -2]"},
		Action:  "added",
	}
	p.On("ObserveFeaturesEventsNearToAdmin", mock.Anything,
		mock.MatchedBy(func(cb func(id string, f model.Feature, action string) error) bool {
			cb(example.AdminID, example.Feature, example.Action)
			return true
		})).Return(nil)

	m.On("Publish", mock.Anything, mock.Anything).Return(nil)

	err := w.Run(context.Background(), nil)
	require.NoError(t, err)

	m.AssertCalled(t, "Publish", mock.AnythingOfType("string"), mock.MatchedBy(func(data []byte) bool {
		actual := &core.FeatureEventsNearToAdminPayload{}
		json.Unmarshal(data, actual)
		return assert.EqualValues(t, example, actual)
	}))
}

func TestObserveFeaturesEventsNearToAdmin(t *testing.T) {
	p := &smocks.PlayerLocationService{}
	m := &smocks.Dispatcher{}
	w := core.NewFeaturesEventsWatcher(m, p)

	ctx, finish := context.WithCancel(context.Background())

	example := &core.FeatureEventsNearToAdminPayload{
		AdminID: "test-admin-id",
		Feature: model.Feature{ID: "test-geofence-id", Group: "geofence", Coordinates: "[-1, -2]"},
		Action:  "added",
	}

	m.On("Subscribe", mock.Anything, mock.Anything, mock.MatchedBy(func(cb func(data []byte) error) bool {
		data, _ := json.Marshal(example)
		cb(data)
		return true
	})).Return(nil)

	actual := &core.FeatureEventsNearToAdminPayload{}
	err := w.OnFeatureEventNearToAdmin(ctx, func(adminID string, feat model.Feature, action string) error {
		actual.AdminID = adminID
		actual.Feature = feat
		actual.Action = action
		finish()
		return nil
	})
	require.NoError(t, err)

	<-ctx.Done()
	assert.EqualValues(t, example, actual)
}
