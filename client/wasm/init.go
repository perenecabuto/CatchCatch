// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"syscall/js"

	"github.com/perenecabuto/CatchCatch/client"
	"github.com/perenecabuto/CatchCatch/server/game"
)

type WASMClient struct {
	client *client.Client
}

func logError(msg string) {
	js.Global().Get("console").Call("error", msg)
}

func callbackFunc(cb func([]js.Value)) js.Callback {
	var callback js.Callback
	callback = js.NewCallback(func(values []js.Value) {
		// defer callback.Release()
		cb(values)
	})
	return callback
}

func registerCallbacks() {
	ws := NewWASMWebSocket()
	cli := client.New(ws)
	ctx := context.Background()
	newPlayer := callbackFunc(func(values []js.Value) {
		if len(values) != 2 {
			js.Global().Call("newPlayer needs addr, callback parmas")
			return
		}

		addr, cb := values[0].String(), values[1]

		player, err := cli.ConnectAsPlayer(ctx, addr)
		if err != nil {
			logError(err.Error())
			return
		}

		playerWrapper := map[string]interface{}{
			"update": callbackFunc(func(vals []js.Value) {
				lat, lon := vals[0].Float(), vals[1].Float()
				err := player.UpdatePlayer(lat, lon)
				if err != nil {
					logError(err.Error())
				}
				return
			}),
			"onRegistered": callbackFunc(func(vals []js.Value) {
				cb := vals[0]
				player.OnRegistered(func(player game.Player) error {
					payload, _ := json.Marshal(player)
					cb.Invoke(js.ValueOf(string(payload)))
					return nil
				})
			}),
			"onGameStarted": callbackFunc(func(vals []js.Value) {
				cb := vals[0]
				player.OnGameStarted(func(game, role string) error {
					cb.Invoke(game, role)
					return nil
				})
			}),
		}

		cb.Invoke(playerWrapper)
	})
	js.Global().Set("catchcatch", map[string]interface{}{
		"NewPlayer": newPlayer,
	})
}

func main() {
	c := make(chan struct{}, 0)

	println("Catchcatch Initialized")

	// register functions
	registerCallbacks()

	readyEvt := js.Global().Get("Event").New("catchcatch:ready")
	js.Global().Get("document").Call("dispatchEvent", readyEvt)

	<-c
}
