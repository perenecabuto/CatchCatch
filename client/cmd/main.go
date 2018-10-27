package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/perenecabuto/CatchCatch/client"
)

var (
	addr = flag.String("addr", "localhost:5000", "http service address")
)

func main() {
	flag.Parse()
	log.SetFlags(0)

	ws := client.NewGorillaWebSocket()
	client := client.New(ws)

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
