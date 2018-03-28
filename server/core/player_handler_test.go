package core

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/mocks"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/websocket"
)

func TestGameWatcher(t *testing.T) {
	wsDriver := new(mocks.WSDriver)
	wss := websocket.NewWSServer(wsDriver)
	c := &mocks.WSConnection{}

	c.On("Send", mock.MatchedBy(func(payload []byte) bool {
		msg := &protobuf.GameInfo{}
		proto.Unmarshal(payload, msg)
		t.Log(msg)
		return true
	})).Return(nil)

	cListener := wss.Add(c)
	playerID := cListener.ID

	gameService := new(mocks.GameService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gameID := "test-gamewatcher-game-1"
	g, _ := game.NewGame(gameID)
	g.SetPlayer(playerID, 0, 0)
	g.Start()
	evt := game.Event{Name: game.GameStarted}

	gameService.On("ObserveGamesEvents", ctx,
		mock.MatchedBy(func(fn func(g *game.Game, evt game.Event) error) bool {
			fn(g, evt)
			return true
		})).Return(nil)

	playerH := NewPlayerHandler(wss, nil, gameService)
	err := playerH.WatchGameEvents(ctx)
	require.NoError(t, err)

	expected, _ := proto.Marshal(&protobuf.GameInfo{
		EventName: proto.String("game:started"),
		Id:        &g.ID, Game: &g.ID,
		Role: proto.String(string("target")),
	})
	c.AssertCalled(t, "Send", expected)
}