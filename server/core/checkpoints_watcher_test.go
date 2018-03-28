package core

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/perenecabuto/CatchCatch/server/mocks"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/websocket"
)

func TestWatchCheckPointsMustNotifyPlayersNearToCheckPoinstsTheDistToIt(t *testing.T) {
	gomega.RegisterTestingT(t)

	wss := websocket.NewWSServer(&mocks.WSDriver{})
	playerService := new(mocks.PlayerLocationService)
	watcher := NewCheckpointWatcher(wss, nil, playerService)

	c := &mocks.WSConnection{}
	c.On("Send", mock.Anything).Return(nil)

	cListener := wss.Add(c)
	playerID := cListener.ID

	distToCheckPoint := 12.0
	checkPoint := model.Feature{Group: "checkpoint", ID: "checkpoint1",
		Coordinates: `{"type": "point", "coordinates": [1, 2]}`}

	playerService.On("ObservePlayerNearToCheckpoint", mock.Anything,
		mock.MatchedBy(func(callback service.PlayerNearToFeatureCallback) bool {
			err := callback(playerID, distToCheckPoint, checkPoint)
			return assert.NoError(t, err)
		})).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := watcher.Run(ctx, nil)
	assert.NoError(t, err)

	expected, _ := proto.Marshal(&protobuf.Detection{
		EventName:    proto.String("checkpoint:detected"),
		Id:           &playerID,
		FeatId:       &checkPoint.ID,
		NearByFeatId: &playerID,
		NearByMeters: &distToCheckPoint,

		Lon: proto.Float64(1), Lat: proto.Float64(2),
	})

	c.AssertCalled(t, "Send", mock.MatchedBy(func(data []byte) bool {
		return assert.Equal(t, data, expected)
	}))
}
