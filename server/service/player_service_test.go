package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service/mocks"
	"github.com/perenecabuto/CatchCatch/server/service/repository"

	"github.com/perenecabuto/CatchCatch/server/service"
)

func TestObserveFeaturesEventsNearToAdmin(t *testing.T) {
	r := &mocks.Repository{}
	s := &mocks.EventStream{}
	pls := service.NewPlayerLocationService(r, s)

	ctx, finish := context.WithCancel(context.Background())
	defer finish()

	adminID := "test-admin-1"

	// fields: ctx, nearByKey, roamKey, roamID, meters, callback
	any := mock.Anything
	s.On("StreamNearByEvents", any, any, any, any, any, any).
		Run(func(args mock.Arguments) {
			nearByKey, cb := args[1].(string), args[5].(repository.DetectionHandler)
			cb(&repository.Detection{NearByFeatID: adminID, FeatID: nearByKey + "-test-1"})
			cb(&repository.Detection{NearByFeatID: adminID, FeatID: nearByKey + "-test-2"})
		}).Return(nil)

	example := map[string]model.Feature{
		"player-test-1":     model.Feature{ID: "player-test-1", Group: "player"},
		"player-test-2":     model.Feature{ID: "player-test-2", Group: "player"},
		"geofences-test-1":  model.Feature{ID: "geofences-test-1", Group: "geofences"},
		"geofences-test-2":  model.Feature{ID: "geofences-test-2", Group: "geofences"},
		"checkpoint-test-1": model.Feature{ID: "checkpoint-test-1", Group: "checkpoint"},
		"checkpoint-test-2": model.Feature{ID: "checkpoint-test-2", Group: "checkpoint"},
	}

	actualFeatures := map[string]model.Feature{}
	var mu sync.RWMutex
	var wg sync.WaitGroup
	wg.Add(len(example))
	err := pls.ObserveFeaturesEventsNearToAdmin(ctx, func(actualID string, f model.Feature, action string) error {
		assert.Equal(t, adminID, actualID)
		mu.Lock()
		actualFeatures[f.ID] = f
		mu.Unlock()
		wg.Done()
		return nil
	})
	require.NoError(t, err)

	wg.Wait()

	assert.EqualValues(t, example, actualFeatures)
}
