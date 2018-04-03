package core_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tidwall/gjson"

	core "github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/model"
	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
)

func TestFeaturesWatcherNotifiesFeaturesEventsToAdmin(t *testing.T) {
	m := new(smocks.Dispatcher)
	p := new(smocks.PlayerLocationService)
	w := core.NewFeaturesEventsWatcher(m, p)

	example := map[string]string{
		"id":          "test-admin-id",
		"featID":      "test-geofence-id",
		"coordinates": "[-1, -2]",
		"group":       "geofences",
		"action":      "set",
	}
	p.On("ObserveFeaturesEventsNearToAdmin", mock.Anything,
		mock.MatchedBy(func(cb func(id string, f model.Feature, action string) error) bool {
			f := model.Feature{
				ID:          example["featID"],
				Group:       example["group"],
				Coordinates: example["coordinates"],
			}
			cb(example["id"], f, example["action"])
			return true
		})).Return(nil)

	m.On("Publish", mock.Anything, mock.Anything).Return(nil)

	err := w.Run(context.Background(), nil)
	require.NoError(t, err)

	m.AssertCalled(t, "Publish", core.FeaturesMessageTopic, mock.MatchedBy(func(data []byte) bool {
		actual := map[string]string{
			"id":          gjson.GetBytes(data, "id").String(),
			"featID":      gjson.GetBytes(data, "featID").String(),
			"coordinates": gjson.GetBytes(data, "coordinates").String(),
			"group":       gjson.GetBytes(data, "group").String(),
			"action":      gjson.GetBytes(data, "action").String(),
		}
		return assert.EqualValues(t, example, actual)
	}))
}
