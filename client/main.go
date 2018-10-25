package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var addr = flag.String("addr", "localhost:5000", "http service address")

// TODO: OnDisconnect
// TODO: add request
//// TODO: SERVER validate step size on server when player is in game
//// TODO: SERVER check rank bug
//// TODO: SERVER bug disconnect admin
// TODO: get/request notifications about games around
// TODO: request game ranking
// TODO: request global ranking
// TODO: request features info (game arena, checkpoint)
// TODO: request how many players are around
// TODO: bin to load player routes from shape file
// TODO: admin client
// TODO: add event to listen for players closer/inside a shape
// TODO: admin event for player connected
// TODO: admin event for player entered into a game

func main() {
	flag.Parse()
	log.SetFlags(0)

	// TODO: Create constructors
	client := &Client{&GorillaWebSocket{}}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGKILL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("connecting to %s", *addr)
	player, err := client.ConnectAsPlayer(ctx, *addr)
	if err != nil {
		log.Fatal("connect:", err)
	}

	log.Printf("connected...")
	err = player.UpdatePlayer(-30.03495, -51.21866)
	if err != nil {
		log.Fatal("update player:", err)
	}

	player.OnGameStarted(func(game, role string) error {
		log.Println("Game:", game, " Started - Role:", role)
		return nil
	})
	player.OnGamePlayerNearToTarget(func(dist float64) error {
		log.Println("near - dist to target:", dist)
		return nil
	})
	player.OnGamePlayerWin(func(dist float64) error {
		log.Println("you win - dist to target:", dist)
		return nil
	})
	player.OnGamePlayerLose(func() error {
		log.Println("you lose")
		return nil
	})

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			coords := player.Coords()
			coords[0] += 0.0001
			coords[1] += 0.0001
			player.UpdatePlayer(coords[0], coords[1])
		case <-interrupt:
			log.Println("closing")
			player.Disconnect()
			cancel()
			return
		}
	}
}
