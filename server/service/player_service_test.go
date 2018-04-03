package service_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/perenecabuto/CatchCatch/server/model"
	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
	"github.com/perenecabuto/CatchCatch/server/service/repository"

	"github.com/perenecabuto/CatchCatch/server/service"
)

func TestObserveFeaturesEventsNearToAdmin(t *testing.T) {
	r := new(smocks.Repository)
	s := &mockStream{}
	pls := service.NewPlayerLocationService(r, s)

	ctx, finish := context.WithCancel(context.Background())
	defer finish()

	adminID := "test-admin-1"

	s.streamNearByEvents = func(ctx context.Context, nearByKey, roamKey, roamID string, meters int, cb repository.DetectionHandler) error {
		cb(&repository.Detection{NearByFeatID: adminID, FeatID: nearByKey + "-test-1"})
		cb(&repository.Detection{NearByFeatID: adminID, FeatID: nearByKey + "-test-2"})
		return nil
	}

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
	err := pls.ObserveFeaturesEventsNearToAdmin(ctx, func(actualID string, f model.Feature, action string) error {
		assert.Equal(t, adminID, actualID)
		mu.Lock()
		actualFeatures[f.ID] = f
		mu.Unlock()
		return nil
	})
	require.NoError(t, err)

	// FIXME: this lock is necessary to test with race condition
	mu.Lock()
	exampleJSON, _ := json.Marshal(example)
	actualJSON, _ := json.Marshal(actualFeatures)
	mu.Unlock()

	assert.Equal(t, string(exampleJSON), string(actualJSON))
}

type mockStream struct {
	streamNearByEvents func(ctx context.Context, nearByKey, roamKey, roamID string, meters int, callback repository.DetectionHandler) error
	streamIntersects   func(ctx context.Context, intersectKey, onKey, onKeyID string, callback repository.DetectionHandler) error
}

func (s mockStream) StreamNearByEvents(ctx context.Context, nearByKey, roamKey, roamID string, meters int, callback repository.DetectionHandler) error {
	return s.streamNearByEvents(ctx, nearByKey, roamKey, roamID, meters, callback)
}

func (s mockStream) StreamIntersects(ctx context.Context, intersectKey, onKey, onKeyID string, callback repository.DetectionHandler) error {
	return nil
}
