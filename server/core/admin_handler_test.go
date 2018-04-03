package core_test

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tidwall/sjson"

	"github.com/stretchr/testify/require"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/websocket"

	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
	wsmocks "github.com/perenecabuto/CatchCatch/server/websocket/mocks"
)

func TestAdminHandlerMustNotifyAboutFeaturesNear(t *testing.T) {
	ws := websocket.NewWSServer(nil)
	p := new(smocks.PlayerLocationService)
	m := new(smocks.Dispatcher)
	h := core.NewAdminHandler(ws, p, m)

	ctx, finish := context.WithCancel(context.Background())

	adminConn := new(wsmocks.WSConnection)
	adminID := ws.Add(adminConn).ID
	adminConn.On("Send", mock.Anything).Return(nil)

	action := "set"
	example := &protobuf.Feature{
		Id:        proto.String("test-geofence-1"),
		Group:     proto.String("geofences"),
		Coords:    proto.String("[[1,2,3], [1,2,3]]"),
		EventName: proto.String("admin:feature:" + action),
	}

	m.On("Subscribe", mock.Anything, mock.Anything, mock.MatchedBy(func(cb func(data []byte) error) bool {
		go func() {
			payload, _ := sjson.SetBytes([]byte{}, "id", adminID)
			payload, _ = sjson.SetBytes(payload, "featID", example.GetId())
			payload, _ = sjson.SetBytes(payload, "group", example.GetGroup())
			payload, _ = sjson.SetBytes(payload, "coordinates", example.GetCoords())
			payload, _ = sjson.SetBytes(payload, "action", action)
			cb(payload)
			finish()
		}()
		return true
	})).Return(nil)

	err := h.WatchFeatureEvents(ctx)
	require.NoError(t, err)

	adminConn.AssertCalled(t, "Send", mock.MatchedBy(func(data []byte) bool {
		actual := &protobuf.Feature{}
		err := proto.Unmarshal(data, actual)
		return assert.NoError(t, err) &&
			assert.EqualValues(t, example, actual)
	}))
}
