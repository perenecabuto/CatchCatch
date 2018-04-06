package core

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/websocket"

	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
	wsmocks "github.com/perenecabuto/CatchCatch/server/websocket/mocks"
)

func TestPlayerHandlerOnStartObserveGameEvents(t *testing.T) {
	wsDriver := new(wsmocks.WSDriver)
	c := new(wsmocks.WSConnection)

	c.On("Send", mock.MatchedBy(func(payload []byte) bool {
		msg := &protobuf.GameInfo{}
		proto.Unmarshal(payload, msg)
		t.Log(msg)
		return true
	})).Return(nil)

	gs := new(smocks.GameService)
	m := new(smocks.Dispatcher)
	w := NewGameWorker(gs, m)
	playerH := NewPlayerHandler(nil, w)
	wss := websocket.NewWSServer(wsDriver, playerH)
	playerID := wss.Add(c).ID

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gameID := "test-gamewatcher-game-1"
	g := game.NewGame(gameID)
	g.SetPlayer(playerID, 0, 0)
	g.Start()

	example := &GameEventPayload{Event: GameStarted, PlayerID: playerID, Game: g}

	m.On("Subscribe", mock.Anything, mock.Anything, mock.MatchedBy(func(cb func(data []byte) error) bool {
		data, _ := json.Marshal(example)
		cb(data)
		return true
	})).Return(nil)

	err := playerH.OnStart(ctx, wss)
	require.NoError(t, err)

	expected, _ := proto.Marshal(&protobuf.GameInfo{
		EventName: proto.String("game:started"),
		Id:        &g.ID, Game: &g.ID,
		Role: proto.String(string("target")),
	})
	c.AssertCalled(t, "Send", expected)
}
