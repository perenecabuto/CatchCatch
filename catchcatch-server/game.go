package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	io "github.com/googollee/go-socket.io"
)

const MinPlayersPerGame = 3

type Game struct {
	ID             string
	players        map[string]*Player
	duration       time.Duration
	started        bool
	targetPlayerID string

	stopFunc context.CancelFunc
}

func NewGame(id string, duration time.Duration) *Game {
	return &Game{ID: id, duration: duration, started: false,
		players: make(map[string]*Player),
	}
}

func (g Game) String() string {
	return fmt.Sprintf("%s(%d)started=%v", g.ID, len(g.players), g.started)
}

func (g *Game) SetPlayer(p *Player) {
	if player, exists := g.players[p.ID]; exists {
		// log.Println("Game:"+g.ID+":update player:", p.ID)
		player.X = p.X
		player.Y = p.Y
	} else {
		// log.Println("Game:"+g.ID+":add player:", p.ID)
		g.players[p.ID] = p
	}
}

func (g *Game) Start() {
	if g.started {
		g.Stop()
	}

	g.started = true
	go func() {
		log.Println("---------------------------")
		log.Println("Game:", g.ID, ":start!!!!!!")
		log.Println("---------------------------")

		g.sortTargetPlayer()

		ctx, cancel := context.WithTimeout(context.Background(), g.duration)
		g.stopFunc = cancel
		<-ctx.Done()
		g.started = false
		g.players = map[string]*Player{}
		g.targetPlayerID = ""
		log.Println("Game:", g.ID, ":stop!")
	}()
}

func (g *Game) Stop() {
	if g.stopFunc != nil {
		g.stopFunc()
	}
}

func (g Game) Started() bool {
	return g.started
}

func (g Game) Ready() bool {
	return !g.started && len(g.players) >= MinPlayersPerGame
}

func (g *Game) WatchPlayers(stream EventStream) {
	go stream.StreamIntersects("player", "geofences", g.ID, func(d *Detection) {
		p := &Player{ID: d.FeatID, X: d.Lat, Y: d.Lon}
		switch d.Intersects {
		case Enter:
			if !g.started {
				log.Println("Game:"+g.ID+":player enter:", p)
			}
			g.addUntilReady(p)
		case Inside:
			if _, exists := g.players[p.ID]; g.started && exists {
				log.Printf("Game:%s:player move:target?%v\n", g.ID, g.targetPlayerID == d.FeatID)
				g.SetPlayer(p)
				return
			}
			g.addUntilReady(p)
		case Exit:
			if _, exists := g.players[p.ID]; !exists {
				return
			}

			delete(g.players, p.ID)
			if !g.started {
				log.Println("Game:"+g.ID+":detect=exit:player", p)
				return
			}

			if len(g.players) == 1 {
				for _, p := range g.players {
					log.Println("Game:"+g.ID+":detect=winner:player", p)
					break
				}
				g.Stop()
			} else if p.ID == g.targetPlayerID {
				log.Println("Game:"+g.ID+":detect=target-player-exit:", p)
				g.Stop()
			} else if len(g.players) == 0 {
				log.Println("Game:"+g.ID+":detect=no-players:player", p)
				g.Stop()
			} else {
				log.Println("Game:"+g.ID+":detect=exit-loose:player", p)
			}
		}
	})
}

func (g *Game) addUntilReady(p *Player) {
	if g.started {
		return
	}
	g.SetPlayer(p)
	if g.Ready() {
		g.Start()
	}
}

func (g *Game) sortTargetPlayer() {
	ids := make([]string, 0)
	for id := range g.players {
		ids = append(ids, id)
	}
	g.targetPlayerID = ids[rand.Intn(len(ids))]
}

var (
	games = make(map[string]*Game)
)

func handleGames(stream EventStream, sessions *SessionManager, service *PlayerLocationService) {
	err := stream.StreamNearByEvents("player", "geofences", 0, func(d *Detection) {
		gameID := d.NearByFeatID
		game, exists := games[gameID]
		if !exists {
			log.Println("Creating game", gameID)
			gameDuration := time.Minute
			game = NewGame(gameID, gameDuration)
			games[gameID] = game
			game.WatchPlayers(stream)
		}
	})
	if err != nil {
		log.Println("Error to stream geofence:event", err)
	}
}

func handleCheckointsDetection(stream EventStream, sessions *SessionManager, server *io.Server) {
	err := stream.StreamNearByEvents("player", "checkpoint", 1000, func(d *Detection) {
		payload, _ := json.Marshal(d)
		if err := sessions.Emit(d.FeatID, "checkpoint:detected", string(payload)); err != nil {
			log.Println("Error to notify player", d.FeatID, err)
		}
		server.BroadcastTo("main", "admin:feature:checkpoint", d)
	})
	if err != nil {
		log.Println("Error to stream geofence:event", err)
	}
}
