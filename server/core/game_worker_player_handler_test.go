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

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/websocket"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestObjectsGraph(t *testing.T) {
	wsDriver := &wsmocks.WSDriver{}
	gs := &smocks.GameService{}
	m := &smocks.Dispatcher{}
	pls := &smocks.PlayerLocationService{}
	gw := core.NewGameWorker(gs, m)
	pw := core.NewPlayersWatcher(m, pls)
	playerH := core.NewPlayerHandler(pls, pw, gw)
	wss := websocket.NewWSServer(wsDriver)
	c := &wsmocks.WSConnection{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	player := &model.Player{ID: "player-test-1"}
	ws := wss.Add(player.ID, c)
	pls.On("GetByID", mock.Anything).Return(player, nil)
	pls.On("SetActive", mock.Anything, mock.Anything).Return(nil)
	pls.On("Set", mock.Anything, mock.Anything).Return(nil)
	c.On("Send", mock.Anything).Return(nil)

	playerH.OnConnection(ctx, ws)

	m.On("Subscribe", mock.Anything, mock.Anything, mock.MatchedBy(func(cb func(data []byte) error) bool {
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
