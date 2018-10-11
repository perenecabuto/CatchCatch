package core

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/service"

	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
)

func TestCheckpointsWatcherWatcherPlayersNearToCheckpoints(t *testing.T) {
	playerService := &smocks.PlayerLocationService{}
	dispatcher := &smocks.Dispatcher{}
	watcher := NewCheckpointWatcher(dispatcher, playerService)

	playerID := "player-test-1"

	distToCheckPoint := 12.0
	checkPoint := model.Feature{Group: "checkpoint", ID: "checkpoint1",
		Coordinates: `{"type": "point", "coordinates": [1, 2]}`}

	dispatcher.On("Publish", mock.Anything, mock.Anything).Return(nil)

	ctx, finish := context.WithCancel(context.Background())

	playerService.On("ObservePlayerNearToCheckpoint", mock.Anything,
		mock.MatchedBy(func(callback service.PlayerNearToFeatureCallback) bool {
			err := callback(playerID, distToCheckPoint, checkPoint)
			finish()
			return assert.NoError(t, err)
		})).Return(nil)

	err := watcher.Run(ctx, nil)
	assert.NoError(t, err)

	evtName := "checkpoint:detected"
	expected, _ := proto.Marshal(&protobuf.Detection{
		EventName:    proto.String(evtName),
		Id:           &playerID,
		FeatId:       &checkPoint.ID,
		NearByFeatId: &playerID,
		NearByMeters: &distToCheckPoint,

		Lon: proto.Float64(1), Lat: proto.Float64(2),
	})

	dispatcher.AssertCalled(t, "Publish", evtName, mock.MatchedBy(func(data []byte) bool {
		return assert.Equal(t, data, expected)
	}))
}

func TestCheckpointsWatcherNotifyThePlayersAroundCheckpoints(t *testing.T) {
	playerService := &smocks.PlayerLocationService{}
	dispatcher := &smocks.Dispatcher{}
	watcher := NewCheckpointWatcher(dispatcher, playerService)

	playerID := "player-test-1"
	ctx, finish := context.WithCancel(context.Background())

	example := &protobuf.Detection{
		EventName:    proto.String("checkpoint:detected"),
		Id:           &playerID,
		FeatId:       proto.String("checkpoint-test-1"),
		NearByFeatId: &playerID,
		NearByMeters: proto.Float64(100),

		Lon: proto.Float64(1), Lat: proto.Float64(2),
	}

	dispatcher.On("Subscribe", ctx, example.GetEventName(),
		mock.MatchedBy(func(cb func([]byte) error) bool {
			data, _ := proto.Marshal(example)
			err := cb(data)
			return assert.NoError(t, err)
		})).Return(nil)

	var actual *protobuf.Detection
	err := watcher.OnCheckpointNearToPlayer(ctx, func(payload *protobuf.Detection) error {
		actual = payload
		finish()
		return nil
	})
	assert.NoError(t, err)

	assert.Equal(t, example, actual)
}
