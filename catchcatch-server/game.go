package main

import (
	"log"
	"math/rand"
	"time"
)

const MIN_PLAYERS_PER_GAME = 2

type Game struct {
	ID             string
	players        map[string]*Player
	duration       time.Duration
	started        bool
	targetPlayerID string
}

func NewGame(id string, duration time.Duration) *Game {
	return &Game{ID: id, duration: duration, started: false,
		players: make(map[string]*Player),
	}
}

func (g Game) SetPlayer(p *Player) {
	if player, exists := g.players[p.ID]; exists {
		log.Println("Update player", p.ID, "to game", g.ID)
		player.X = p.X
		player.Y = p.Y
	} else {
		log.Println("AddPlayer", p.ID, "to game", g.ID)
		g.players[p.ID] = p
	}
}

func (g *Game) Start() {
	g.started = true
	g.sortTargetPlayer()
	go g.startTimer()
}

func (g *Game) WatchPlayers(stream EventStream) {
	go stream.StreamIntersects("player", "geofences", g.ID, func(d *Detection) {
		log.Println("Game player detected", d, g.targetPlayerID, g.targetPlayerID == d.FeatID)
	})
}

func (g *Game) sortTargetPlayer() {
	ids := make([]string, 0)
	for id := range g.players {
		ids = append(ids, id)
	}
	g.targetPlayerID = ids[rand.Intn(len(ids))]
}

func (g *Game) startTimer() {
	log.Println("startTimer", "start")
	t := time.NewTimer(g.duration)
	<-t.C
	g.started = false
	log.Println("startTimer", "stop")
}

func (g Game) Started() bool {
	return g.started
}

func (g Game) HasPlayer(p *Player) bool {
	return g.players[p.ID] != nil
}

func (g Game) Ready() bool {
	return len(g.players) >= MIN_PLAYERS_PER_GAME
}

var (
	games = make(map[string]*Game)
)

func handleGames(stream EventStream, sessions *SessionManager, service *PlayerLocationService) {
	err := stream.StreamNearByEvents("player", "geofences", 0, func(d *Detection) {
		gameID := d.NearByFeatID
		p, err := service.PlayerById(d.FeatID)
		if err != nil {
			log.Println("Error starting and adding player", d.FeatID, "to game", gameID, err)
			return
		}
		game, exists := games[gameID]
		if !exists {
			log.Println("Creating game", gameID)
			gameDuration := time.Minute
			game = NewGame(gameID, gameDuration)
			games[gameID] = game
			game.WatchPlayers(stream)
		} else if !game.Started() {
			game.SetPlayer(p)
			if game.Ready() {
				game.Start()
			}
		}

	})
	if err != nil {
		log.Println("Error to stream geofence:event", err)
	}
}
