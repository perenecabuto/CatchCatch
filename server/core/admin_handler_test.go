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
	p := &smocks.PlayerLocationService{}
	w := &wmocks.EventsNearToAdminWatcher{}
	h := core.NewAdminHandler(p, w)
	ctx, finish := context.WithCancel(context.Background())
	defer finish()

	adminConn := &wsmocks.WSConnection{}
	adminConn.On("Send", mock.Anything).Return(nil)
	wss := websocket.NewWSServer(nil)
	adminID := "admin-test-1"
	wss.Add(adminID, adminConn)

	action := "added"
	example := &protobuf.Feature{
		Id:        "test-geofence-1",
		Group:     "geofences",
		Coords:    "[[1,2,3], [1,2,3]]",
		EventName: "admin:feature:" + action,
	}

	w.On("OnFeatureEventNearToAdmin", ctx,
		mock.MatchedBy(func(cb func(string, model.Feature, string) error) bool {
			f := model.Feature{ID: example.GetId(), Coordinates: example.GetCoords(), Group: example.GetGroup()}
			cb(adminID, f, action)
			return true
		})).Return(nil)

	h.OnStart(ctx, wss)

	adminConn.AssertCalled(t, "Send", mock.MatchedBy(func(data []byte) bool {
		actual := &protobuf.Feature{}
		err := proto.Unmarshal(data, actual)
		return assert.NoError(t, err) &&
			assert.EqualValues(t, example, actual)
	}))
}
