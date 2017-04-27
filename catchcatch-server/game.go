package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"time"
)

// MinPlayersPerGame ...
const MinPlayersPerGame = 3

// Game controls rounds and players
type Game struct {
	ID           string
	players      map[string]*Player
	duration     time.Duration
	started      bool
	targetPlayer *Player

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
	log.Println("game:", g.ID, ":start!!!!!!")
	log.Println("---------------------------")
	g.sortTargetPlayer()
	for id := range g.players {
		role := "hunter"
		if id == g.targetPlayer.ID {
			role = "target"
		}
		sessions.Emit(id, "game:started", &GameInfo{Game: g.ID, Role: role})
	}
	g.started = true

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), g.duration)
		g.stopFunc = cancel

		<-ctx.Done()
		log.Println("---------------------------")
		log.Println("game:", g.ID, ":stop!!!!!!!")
		log.Println("---------------------------")
		if _, exists := g.players[g.targetPlayer.ID]; exists {
			go sessions.Emit(g.targetPlayer.ID, "game:target:win", "")
		}
		sessions.BroadcastTo(g.playerIDs(), "game:finish", g.rank())

		g.started = false
		g.players = make(map[string]*Player)
		g.targetPlayer = nil
	}()
}

// GameInfo ...
type GameInfo struct {
	Role string `json:"role"`
	Game string `json:"game"`
}

// PlayerRank ...
type PlayerRank struct {
	Player string `json:"player"`
	Points int    `json:"points"`
}

// GameRank ...
type GameRank struct {
	Game       string       `json:"game"`
	PlayerRank []PlayerRank `json:"points_per_player"`
}

func (g *Game) rank() *GameRank {
	rank := &GameRank{Game: g.ID, PlayerRank: make([]PlayerRank, 0)}

	playersDistToTarget := map[int]*Player{}
	for _, p := range g.players {
		dist := p.DistTo(g.targetPlayer)
		playersDistToTarget[int(dist)] = p
	}
	dists := make([]int, 0)
	for dist := range playersDistToTarget {
		dists = append(dists, dist)
	}
	sort.Ints(dists)

	maxDist := dists[len(dists)-1] + 1
	for _, dist := range dists {
		p := playersDistToTarget[dist]
		points := 100 * (maxDist - dist) / maxDist
		rank.PlayerRank = append(rank.PlayerRank, PlayerRank{Player: p.ID, Points: points})
	}

	return rank
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

func (g *Game) SetPlayerUntilReady(p *Player, sessions *SessionManager) {
	if g.started {
		if g.HasPlayer(p.ID) {
			g.updateAndNofityPlayer(p, sessions)
		}
		return
	}
	if _, exists := g.players[p.ID]; !exists {
		log.Printf("game:%s:detect=enter:%s\n", g.ID, p.ID)
	}
	g.SetPlayer(p)
	if g.Ready() {
		g.Start(sessions)
	}
}

func (g *Game) updateAndNofityPlayer(p *Player, sessions *SessionManager) {
	if _, exists := g.players[p.ID]; !exists {
		return
	}
	g.SetPlayer(p)
	if p.ID == g.targetPlayer.ID {
		return
	}
	dist := p.DistTo(g.targetPlayer)
	if dist <= 20 {
		log.Printf("game:%s:detect=winner:%s:dist:%f\n", g.ID, p.ID, dist)
		sessions.Emit(g.targetPlayer.ID, "game:loose", g.ID)
		delete(g.players, g.targetPlayer.ID)
		go sessions.Emit(p.ID, "game:target:reached", strconv.FormatFloat(dist, 'f', 0, 64))
		g.Stop()
	} else if dist <= 100 {
		log.Printf("game:%s:detect=near:%s:dist:%f\n", g.ID, p.ID, dist)
		sessions.Emit(p.ID, "game:target:near", strconv.FormatFloat(dist, 'f', 0, 64))
		// } else {
		// log.Printf("game:%s:detect=far:%s:dist:%f\n", g.ID, p.ID, dist)
	}
}

func (g *Game) SetPlayer(p *Player) {
	if player, exists := g.players[p.ID]; exists {
		player.Lon = p.Lon
		player.Lat = p.Lat
	} else {
		g.players[p.ID] = p
	}
}

func (g *Game) RemovePlayer(p *Player, sessions *SessionManager) {
	if _, exists := g.players[p.ID]; !exists {
		return
	}

	delete(g.players, p.ID)
	if !g.started {
		log.Println("game:"+g.ID+":detect=exit:", p)
		return
	}

	if len(g.players) == 1 {
		log.Println("game:"+g.ID+":detect=last-one:", p)
		g.Stop()
	} else if p.ID == g.targetPlayer.ID {
		log.Println("game:"+g.ID+":detect=target-loose:", p)
		sessions.Emit(p.ID, "game:loose", g.ID)
		g.Stop()
	} else if len(g.players) == 0 {
		log.Println("game:"+g.ID+":detect=no-players:", p)
		sessions.Emit(p.ID, "game:finish", g.ID)
		g.Stop()
	} else {
		log.Println("game:"+g.ID+":detect=loose:", p)
		sessions.Emit(p.ID, "game:loose", g.ID)
	}
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
	g.targetPlayer = g.players[randPlayerID]
}
