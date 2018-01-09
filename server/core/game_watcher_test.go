package core

import (
	"testing"

	"github.com/perenecabuto/CatchCatch/server/mocks"
	"github.com/perenecabuto/CatchCatch/server/websocket"
)

func TestGameWatcher(t *testing.T) {
	wss := &websocket.WSServer{}
	gameService := &mocks.GameService{}
	serverID := "test-gamewatcher-server-1"
	NewGameWatcher(serverID, gameService, wss)
}
