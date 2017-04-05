package main

import (
	"log"
)

type Game struct {
	ID      string
	started bool
	players map[string]*Player
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
}

const MIN_PLAYERS_PER_GAME = 2

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
			game = &Game{gameID, false, make(map[string]*Player)}
			games[gameID] = game
		} else if !game.Started() && game.Ready() {
			game.Start()
		} else if !game.HasPlayer(p) {
			log.Println("Ignoring player", p.ID, "game", gameID, "is already started")
			return
		}

		log.Println("game started:", game.Started())
		game.SetPlayer(p)
	})
	if err != nil {
		log.Println("Error to stream geofence:event", err)
	}
}
