package core_test

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/websocket"

	wmocks "github.com/perenecabuto/CatchCatch/server/core/mocks"
	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
	wsmocks "github.com/perenecabuto/CatchCatch/server/websocket/mocks"
)

func TestAdminHandlerMustNotifyAboutFeaturesNear(t *testing.T) {
	ws := websocket.NewWSServer(nil)
	p := new(smocks.PlayerLocationService)
	w := new(wmocks.EventsNearToAdminWatcher)
	h := core.NewAdminHandler(ws, p, w)

	ctx, finish := context.WithCancel(context.Background())

	adminConn := new(wsmocks.WSConnection)
	adminID := ws.Add(adminConn).ID
	adminConn.On("Send", mock.Anything).Return(nil)

	action := "added"
	example := &protobuf.Feature{
		Id:        proto.String("test-geofence-1"),
		Group:     proto.String("geofences"),
		Coords:    proto.String("[[1,2,3], [1,2,3]]"),
		EventName: proto.String("admin:feature:" + action),
	}

	w.On("ObserveFeaturesEventsNearToAdmin", ctx,
		mock.MatchedBy(func(cb func(string, model.Feature, string) error) bool {
			f := model.Feature{ID: example.GetId(), Coordinates: example.GetCoords(), Group: example.GetGroup()}
			cb(adminID, f, action)
			finish()
			return true
		})).Return(nil)

	h.WatchFeatureEvents(ctx)

	adminConn.AssertCalled(t, "Send", mock.MatchedBy(func(data []byte) bool {
		actual := &protobuf.Feature{}
		err := proto.Unmarshal(data, actual)
		return assert.NoError(t, err) &&
			assert.EqualValues(t, example, actual)
	}))
}
