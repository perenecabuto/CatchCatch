package core_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	cupaloy "github.com/bradleyjkemp/cupaloy"
	memviz "github.com/bradleyjkemp/memviz"

	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
	wsmocks "github.com/perenecabuto/CatchCatch/server/websocket/mocks"
	wmocks "github.com/perenecabuto/CatchCatch/server/worker/mocks"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/websocket"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestObjectsGraph(t *testing.T) {
	pls := &smocks.PlayerLocationService{}
	gs := &smocks.GameService{}
	d := &smocks.Dispatcher{}
	manager := &wmocks.Manager{}

	gameworker := core.NewGameWorker(gs, d)
	geofences := core.NewGeofenceEventsWorker(pls, manager, d)
	playerH := core.NewPlayerHandler(pls, gameworker, geofences)
	wsDriver := &wsmocks.WSDriver{}

	wss := websocket.NewWSServer(wsDriver, playerH)
	c := &wsmocks.WSConnection{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pls.On("Set", mock.Anything).Return(nil)
	c.On("Send", mock.Anything).Return(nil)

	ws := wss.Add(c)
	playerH.OnConnection(ctx, ws)

	d.On("Subscribe", mock.Anything, mock.Anything, mock.MatchedBy(func(cb func(data []byte) error) bool {
		data, err := json.Marshal(&core.GameEventPayload{
			PlayerID: ws.ID,
			Event:    core.GamePlayerLose, Game: "game-id-5",
			PlayerRole: game.GameRoleHunter, DistToTarget: 67,
		})
		require.NoError(t, err)
		cb(data)
		return true
	})).Return(nil)

	err := playerH.OnStart(ctx, wss)
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	memviz.Map(buf, playerH)
	fmt.Println(buf.String())
	cupaloy.SnapshotT(t, buf.Bytes())
}
