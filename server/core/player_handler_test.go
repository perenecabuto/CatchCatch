package core_test

import (
	"context"
	"encoding/json"
	"log"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/service/messages"
	"github.com/perenecabuto/CatchCatch/server/websocket"

	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
	wsmocks "github.com/perenecabuto/CatchCatch/server/websocket/mocks"
)

func TestPlayerHandlerObeserveAndNotifyPlayerNearToTargetEvent(t *testing.T) {
	tests := []struct {
		payload *core.GameEventPayload
		proto   proto.Message
	}{
		{
			&core.GameEventPayload{
				Event: core.GameStarted, Game: "game-id", PlayerRole: game.GameRoleTarget,
			},
			&protobuf.GameInfo{
				EventName: core.GameStarted.String(),
				Id:        "game-id", Game: "game-id",
				Role: game.GameRoleTarget.String(),
			},
		},
		{
			&core.GameEventPayload{
				Event: core.GamePlayerNearToTarget, Game: "game-id-2",
				PlayerRole: game.GameRoleHunter, DistToTarget: 100,
			},
			&protobuf.Distance{
				Id:        "game-id-2",
				EventName: core.GamePlayerNearToTarget.String(),
				Dist:      100,
			},
		},
		{
			&core.GameEventPayload{
				Event: core.GamePlayerWin, Game: "game-id-3",
				PlayerRole: game.GameRoleTarget, DistToTarget: 0,
			},
			&protobuf.Distance{
				Id:        "game-id-3",
				EventName: core.GamePlayerWin.String(),
				Dist:      0,
			},
		},
		{
			&core.GameEventPayload{
				Event: core.GamePlayerWin, Game: "game-id-4",
				PlayerRole: game.GameRoleHunter, DistToTarget: 4.3,
			},
			&protobuf.Distance{
				Id:        "game-id-4",
				EventName: core.GamePlayerWin.String(),
				Dist:      4.3,
			},
		},
		{
			&core.GameEventPayload{
				Event: core.GamePlayerLose, Game: "game-id-5",
				PlayerRole: game.GameRoleHunter, DistToTarget: 67,
			},
			&protobuf.Simple{
				Id:        "game-id-5",
				EventName: core.GamePlayerLose.String(),
			},
		},
	}

	for _, tt := range tests {
		wsDriver := &wsmocks.WSDriver{}
		gs := &smocks.GameService{}
		m := &smocks.Dispatcher{}
		pls := &smocks.PlayerLocationService{}
		gw := core.NewGameWorker(gs, m)
		pw := core.NewPlayersWatcher(m, pls)
		wss := websocket.NewWSServer(wsDriver)
		c := &wsmocks.WSConnection{}

		playerH := core.NewPlayerHandler(pls, pw, gw)

		expected, err := proto.Marshal(tt.proto)
		require.NoError(t, err)

		c.On("Send", mock.Anything).Run(func(args mock.Arguments) {
			msg := args[0].([]byte)
			log.Println("Send:expected", string(expected))
			log.Println("Send:actual", string(msg))
		}).Return(nil)

		wss.Add(tt.payload.PlayerID, c)
		m.On("Subscribe", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			topic, cb := args[1].(string), args[2].(messages.OnMessage)
			if topic == core.GameWorkerEventsTopic {
				data, _ := json.Marshal(tt.payload)
				cb(data)
			}
		}).Return(nil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = playerH.OnStart(ctx, wss)
		require.NoError(t, err)

		c.AssertCalled(t, "Send", expected)
	}
}

func TestPlayerHandlerSendRankOnGameFinished(t *testing.T) {
	wsDriver := &wsmocks.WSDriver{}
	gs := &smocks.GameService{}
	m := &smocks.Dispatcher{}
	pls := &smocks.PlayerLocationService{}
	gw := core.NewGameWorker(gs, m)
	pw := core.NewPlayersWatcher(m, pls)
	wss := websocket.NewWSServer(wsDriver)
	playerH := core.NewPlayerHandler(pls, pw, gw)

	gameID := "player-handler-game-1"
	players := []game.Player{
		game.Player{Player: model.Player{ID: "player-test-3", Lat: 1, Lon: 2}, Role: "hunter"},
		game.Player{Player: model.Player{ID: "player-test-2", Lat: 1, Lon: 3}, Role: "hunter"},
		game.Player{Player: model.Player{ID: "player-test-1", Lat: 1, Lon: 1}, Role: "target"},
	}

	connections := make([]*wsmocks.WSConnection, len(players))
	for i, p := range players {
		c := &wsmocks.WSConnection{}
		c.On("Send", mock.Anything).Return(nil)
		connections[i] = c
		wss.Add(p.ID, c)
	}
	g := game.NewGameWithParams(gameID, true, players, players[2].ID)

	m.On("Subscribe", mock.Anything, mock.Anything, mock.MatchedBy(func(cb func(data []byte) error) bool {
		for _, p := range players {
			data, _ := json.Marshal(&core.GameEventPayload{
				Event: core.GameFinished, PlayerID: p.ID, PlayerRole: p.Role,
				Game: g.ID, Rank: g.Rank()})
			cb(data)
		}
		return true
	})).Return(nil)

	ctx := context.Background()
	err := playerH.OnStart(ctx, wss)
	require.NoError(t, err)

	for _, c := range connections {
		playerRank := g.Rank().PlayerRank
		rank := make([]*protobuf.PlayerRank, len(playerRank))
		for i, pr := range playerRank {
			rank[i] = &protobuf.PlayerRank{Player: pr.Player.ID, Points: int32(pr.Points)}
		}
		expected, _ := proto.Marshal(&protobuf.GameRank{
			EventName: core.GameFinished.String(),
			Id:        gameID, Game: gameID,
			PlayersRank: rank,
		})
		c.AssertCalled(t, "Send", expected)
	}
}

func TestPlayerHandlerDisconnectDeletedPlayers(t *testing.T) {
	wsDriver := &wsmocks.WSDriver{}
	gs := &smocks.GameService{}
	m := &smocks.Dispatcher{}
	pls := &smocks.PlayerLocationService{}
	gw := core.NewGameWorker(gs, m)
	pw := core.NewPlayersWatcher(m, pls)
	wss := websocket.NewWSServer(wsDriver)
	playerH := core.NewPlayerHandler(pls, pw, gw)

	c := &wsmocks.WSConnection{}
	c.On("Close").Return(nil)

	player := &model.Player{ID: "deleted-player-test"}
	wss.Add(player.ID, c)

	m.On("Subscribe", any, any, any).Run(func(args mock.Arguments) {
		evt, cb := args[1].(string), args[2].(messages.OnMessage)
		if evt == core.PlayerWatcherEventDel {
			data, _ := json.Marshal(player)
			cb(data)
		}
	}).Return(nil)

	ctx := context.Background()
	err := playerH.OnStart(ctx, wss)
	require.NoError(t, err)

	c.AssertCalled(t, "Close")
}
