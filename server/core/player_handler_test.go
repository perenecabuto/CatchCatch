package core

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/websocket"

	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
	wsmocks "github.com/perenecabuto/CatchCatch/server/websocket/mocks"
)

func TestPlayerHandlerOnStartObserveGameEvents(t *testing.T) {
	wsDriver := new(wsmocks.WSDriver)
	c := new(wsmocks.WSConnection)
	c.On("Send", mock.Anything).Return(nil)

	gs := new(smocks.GameService)
	m := new(smocks.Dispatcher)
	w := NewGameWorker(gs, m)
	playerH := NewPlayerHandler(nil, w)
	wss := websocket.NewWSServer(wsDriver, playerH)
	playerID := wss.Add(c).ID

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gameID := "player-handler-game-1"
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

func TestPlayerHandlerSendRankOnGameFinished(t *testing.T) {
	wsDriver := new(wsmocks.WSDriver)
	gs := new(smocks.GameService)
	m := new(smocks.Dispatcher)
	w := NewGameWorker(gs, m)
	playerH := NewPlayerHandler(nil, w)
	wss := websocket.NewWSServer(wsDriver, playerH)

	c1 := new(wsmocks.WSConnection)
	c2 := new(wsmocks.WSConnection)
	c3 := new(wsmocks.WSConnection)
	c1.On("Send", mock.Anything).Return(nil)
	c2.On("Send", mock.Anything).Return(nil)
	c3.On("Send", mock.Anything).Return(nil)
	player1ID := wss.Add(c1).ID
	player2ID := wss.Add(c2).ID
	player3ID := wss.Add(c3).ID

	gameID := "player-handler-game-1"
	players := []game.Player{
		game.Player{Player: model.Player{ID: player3ID, Lat: 1, Lon: 2}, Role: "hunter"},
		game.Player{Player: model.Player{ID: player2ID, Lat: 1, Lon: 3}, Role: "hunter"},
		game.Player{Player: model.Player{ID: player1ID, Lat: 1, Lon: 1}, Role: "target"},
	}
	g := game.NewGameWithParams(gameID, true, players, player3ID)

	m.On("Subscribe", mock.Anything, mock.Anything, mock.MatchedBy(func(cb func(data []byte) error) bool {
		data, _ := json.Marshal(&GameEventPayload{Event: GameFinished, PlayerID: player1ID, Game: g})
		cb(data)
		return true
	})).Return(nil)

	ctx := context.Background()
	err := playerH.OnStart(ctx, wss)
	require.NoError(t, err)

	rank := g.Rank()
	playersRank := make([]*protobuf.PlayerRank, len(rank.PlayerRank))
	for i, pr := range rank.PlayerRank {
		playersRank[i] = &protobuf.PlayerRank{Player: proto.String(pr.Player), Points: proto.Int32(int32(pr.Points))}
	}
	expected := &protobuf.GameRank{
		EventName: proto.String(GameFinished.String()),
		Id:        &rank.Game, Game: &rank.Game,
		PlayersRank: playersRank,
	}

	connections := []*wsmocks.WSConnection{c1, c2, c3}
	for _, c := range connections {
		c.AssertCalled(t, "Send", mock.MatchedBy(func(data []byte) bool {
			actual := &protobuf.GameRank{}
			proto.Unmarshal(data, actual)
			return assert.EqualValues(t,
				expected.PlayersRank, actual.PlayersRank)
		}))
	}
}
