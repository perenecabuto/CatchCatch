package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	io "github.com/googollee/go-socket.io"
)

// MinPlayersPerGame ...
const MinPlayersPerGame = 3

// Game controls rounds and players
type Game struct {
	ID             string
	players        map[string]*Player
	duration       time.Duration
	started        bool
	targetPlayerID string

	stopFunc context.CancelFunc
}

// NewGame create a game with duration
func NewGame(id string, duration time.Duration) *Game {
	return &Game{ID: id, duration: duration, started: false,
		players: make(map[string]*Player)}
}

func (g Game) String() string {
	return fmt.Sprintf("%s(%d)started=%v", g.ID, len(g.players), g.started)
}

// Start the game
func (g *Game) Start(sessions *SessionManager) {
	if g.started {
		g.Stop()
	}

	log.Println("---------------------------")
	log.Println("Game:", g.ID, ":start!!!!!!")
	log.Println("---------------------------")
	g.sortTargetPlayer()
	g.started = true

	for _, p := range g.players {
		if err := sessions.Emit(p.ID, "game:started", `"`+g.ID+`"`); err != nil {
			log.Println("error to emit game:started", p.ID, err)
		}
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), g.duration)
		g.stopFunc = cancel

		<-ctx.Done()
		log.Println("---------------------------")
		log.Println("Game:", g.ID, ":stop!!!!!!")
		log.Println("---------------------------")
		for _, p := range g.players {
			if err := sessions.Emit(p.ID, "game:finish", `"`+g.ID+`"`); err != nil {
				log.Println("error to emit game:started", p.ID, err)
			}
		}

		g.started = false
		g.players = make(map[string]*Player)
		g.targetPlayerID = ""
	}()
}

// Stop a running game
func (g *Game) Stop() {
	if g.stopFunc != nil {
		g.stopFunc()
	}
}

// Started true when game started
func (g Game) Started() bool {
	return g.started
}

// Ready returns true when game is ready to start
func (g Game) Ready() bool {
	return !g.started && len(g.players) >= MinPlayersPerGame
}

// WatchPlayers events
func (g *Game) WatchPlayers(stream EventStream, sessions *SessionManager) {
	go stream.StreamIntersects("player", "geofences", g.ID, func(d *Detection) {
		p := &Player{ID: d.FeatID, X: d.Lat, Y: d.Lon}
		switch d.Intersects {
		case Enter:
			g.setPlayerUntilReady(p, sessions)
		case Exit:
			g.removePlayer(p, sessions)
		case Inside:
			if !g.started {
				g.setPlayerUntilReady(p, sessions)
			} else if g.hasPlayer(p.ID) {
				g.setPlayer(p)
				if p.ID != g.targetPlayerID {
					g.updateAndNofityPlayer(p, sessions)
				} else {
					log.Printf("Game:%s:target:move", g.ID)
				}
			}
		}
	})
}

func (g *Game) setPlayerUntilReady(p *Player, sessions *SessionManager) {
	if g.started {
		return
	}
	if !g.hasPlayer(p.ID) {
		log.Println("Game:"+g.ID+":detect=enter:", p)
	}
	g.setPlayer(p)
	if g.Ready() {
		g.Start(sessions)
	}
}

func (g *Game) updateAndNofityPlayer(p *Player, sessions *SessionManager) {
	targetPlayer, exists := g.players[g.targetPlayerID]
	if !exists {
		log.Printf("Game:%s:move error:target player missing\n", g.ID)
		g.Stop()
		return
	}

	dist := p.DistTo(targetPlayer)
	if dist <= 20 {
		log.Printf("Game:%s:detect=winner:%s:dist:%f\n", g.ID, p.ID, dist)
		sessions.Emit(p.ID, "target:reached", strconv.FormatFloat(dist, 'f', 0, 64))
		g.Stop()
	} else if dist <= 100 {
		sessions.Emit(p.ID, "target:near", strconv.FormatFloat(dist, 'f', 0, 64))
		log.Printf("Game:%s:detect=near:%s:dist:%f\n", g.ID, p.ID, dist)
	} else {
		log.Printf("Game:%s:detect=far:%s:dist:%f\n", g.ID, p.ID, dist)
	}
}

func (g *Game) setPlayer(p *Player) {
	if player, exists := g.players[p.ID]; exists {
		player.X = p.X
		player.Y = p.Y
	} else {
		g.players[p.ID] = p
	}
}

func (g *Game) removePlayer(p *Player, sessions *SessionManager) {
	if !g.hasPlayer(p.ID) {
		return
	}

	delete(g.players, p.ID)
	if !g.started {
		log.Println("Game:"+g.ID+":detect=exit:", p)
		return
	}

	if len(g.players) == 1 {
		for id := range g.players {
			log.Println("Game:"+g.ID+":detect=winner:", id)
			break
		}
		g.Stop()
	} else if p.ID == g.targetPlayerID {
		log.Println("Game:"+g.ID+":detect=target-loose:", p)
		g.Stop()
	} else if len(g.players) == 0 {
		log.Println("Game:"+g.ID+":detect=no-players:", p)
		g.Stop()
	} else {
		log.Println("Game:"+g.ID+":detect=loose:", p)
		sessions.Emit(p.ID, "game:loose", "{}")
	}
}

func (g *Game) hasPlayer(id string) bool {
	_, exists := g.players[id]
	return exists
}

func (g *Game) playerIDs() []string {
	ids := make([]string, 0)
	for id := range g.players {
		ids = append(ids, id)
	}
	return ids
}

func (g *Game) sortTargetPlayer() {
	ids := g.playerIDs()
	randPlayerID := ids[rand.Intn(len(ids))]
	g.targetPlayerID = g.players[randPlayerID].ID
}

func handleGames(stream EventStream, sessions *SessionManager) {
	games := make(map[string]*Game)
	err := stream.StreamNearByEvents("player", "geofences", 0, func(d *Detection) {
		gameID := d.NearByFeatID
		game, exists := games[gameID]
		if !exists {
			log.Println("Creating game", gameID)
			gameDuration := time.Minute
			game = NewGame(gameID, gameDuration)
			games[gameID] = game
			game.WatchPlayers(stream, sessions)
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
