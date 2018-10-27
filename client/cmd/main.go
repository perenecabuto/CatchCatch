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
	"github.com/tkrajina/gpxgo/gpx"
)

var (
	gpxfile      = flag.String("gpx", "", "gpx file <mandatory>")
	addr         = flag.String("addr", "localhost:5000", "http service address")
	stepInterval = flag.Duration("step-interval", time.Millisecond*300,
		"step interval to walk to next point in the gpx file")
)

func main() {
	flag.Parse()

	gpxData, err := gpx.ParseFile(*gpxfile)
	if err != nil {
		flag.Usage()
		log.Fatal(err)
	}

	ws := client.NewGorillaWebSocket()
	cli := client.New(ws)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGKILL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("connecting to %s", *addr)
	player, err := cli.ConnectAsPlayer(ctx, *addr)
	if err != nil {
		log.Fatal("connect:", err)
	}
	log.Printf("connected...")

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
	player.OnGameFinished(func(game string, rank client.Rank) error {
		log.Printf("game:%s finished - rank:%+v", game, rank)
		return nil
	})
	player.OnDisconnect(func() error {
		interrupt <- syscall.SIGABRT
		return nil
	})

	stepChan := make(chan gpx.Point)
	go func() {
		points := pointsFromGPX(gpxData)
		for {
			for _, p := range points {
				select {
				case stepChan <- p:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	for {
		select {
		case point := <-stepChan:
			player.UpdatePlayer(point.Latitude, point.Longitude)
			time.Sleep(*stepInterval)
		case <-interrupt:
			log.Println("closing")
			player.Disconnect()
			return
		}
	}
}

func pointsFromGPX(data *gpx.GPX) []gpx.Point {
	points := make([]gpx.Point, 0)
	for _, track := range data.Tracks {
		for _, segment := range track.Segments {
			for _, p := range segment.Points {
				points = append(points, p.Point)
			}
		}
	}
	return points
}
