// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"log"
	"syscall/js"

	"github.com/perenecabuto/CatchCatch/client"
	"github.com/perenecabuto/CatchCatch/server/game"
)

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

type wasmLogWritter struct{}

func (wlw *wasmLogWritter) Write(p []byte) (n int, err error) {
	text := js.Global().Get("document").Call("getElementById", "log").Get("innerHTML").String()
	js.Global().Get("document").Call("getElementById", "log").Set("innerHTML", string(p)+"\n<hr />\n"+text)
	return len(p), nil
}

func registerCallbacks() {
	ws := NewWASMWebSocket()
	cli := client.New(ws)
	ctx := context.Background()
	newPlayer := callbackFunc(func(values []js.Value) {
		log.SetOutput(&wasmLogWritter{})

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
			// (p *Player) Disconnect() error {
			"disconnect": callbackFunc(func(vals []js.Value) {
				err := player.Disconnect()
				if err != nil {
					logError(err.Error())
				}
			}),
			// (p *Player) UpdatePlayer(lat, lon float64) error {
			"update": callbackFunc(func(vals []js.Value) {
				lat, lon := vals[0].Float(), vals[1].Float()
				err := player.UpdatePlayer(lat, lon)
				if err != nil {
					logError(err.Error())
				}
			}),
			// (p *Player) Coords() LatLon {
			"coords": callbackFunc(func(vals []js.Value) {
				cb := vals[0]
				latlon := player.Coords()
				data, _ := json.Marshal(&latlon)
				cb.Invoke(string(data))
			}),
			// (p *Player) OnRegistered(fn func(player game.Player) error) {
			"onRegistered": callbackFunc(func(vals []js.Value) {
				cb := vals[0]
				player.OnRegistered(func(player game.Player) error {
					payload, _ := json.Marshal(&player)
					cb.Invoke(js.ValueOf(string(payload)))
					return nil
				})
			}),
			// (p *Player) OnGameStarted(fn func(game, role string) error) {
			"onGameStarted": callbackFunc(func(vals []js.Value) {
				cb := vals[0]
				player.OnGameStarted(func(game, role string) error {
					cb.Invoke(game, role)
					return nil
				})
			}),
			// (p *Player) OnGamePlayerNearToTarget(fn func(dist float64) error) {
			"onGamePlayerNearToTarget": callbackFunc(func(vals []js.Value) {
				cb := vals[0]
				player.OnGamePlayerNearToTarget(func(dist float64) error {
					cb.Invoke(dist)
					return nil
				})
			}),
			// (p *Player) OnGamePlayerLose(fn func() error) {
			"onGamePlayerLose": callbackFunc(func(vals []js.Value) {
				cb := vals[0]
				player.OnGamePlayerLose(func() error {
					cb.Invoke()
					return nil
				})
			}),
			// (p *Player) OnGamePlayerWin(fn func(dist float64) error) {
			"onGamePlayerWin": callbackFunc(func(vals []js.Value) {
				cb := vals[0]
				player.OnGamePlayerWin(func(dist float64) error {
					cb.Invoke(dist)
					return nil
				})
			}),
			// (p *Player) OnGameFinished(fn func(game string, rank Rank) error) {
			"onGameFinished": callbackFunc(func(vals []js.Value) {
				cb := vals[0]
				player.OnGameFinished(func(game string, rank client.Rank) error {
					data, _ := json.Marshal(&rank)
					cb.Invoke(game, string(data))
					return nil
				})
			}),
			// (p *Player) OnDisconnect(fn func() error) {
			"onDisconnect": callbackFunc(func(vals []js.Value) {
				cb := vals[0]
				player.OnDisconnect(func() error {
					cb.Invoke()
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
