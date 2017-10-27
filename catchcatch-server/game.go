package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	"github.com/perenecabuto/CatchCatch/catchcatch-server/model"
)

var (
	// ErrAlreadyStarted happens when an action is denied on running game
	ErrAlreadyStarted = errors.New("game already started")
	// ErrPlayerIsNotInTheGame happens when try to change or remove an player not in the game
	ErrPlayerIsNotInTheGame = errors.New("player is not in this game")
)

// GameEvents interface for communication with external game watcher
type GameEvents interface {
	OnGameStarted(g *Game, p GamePlayer)
	OnTargetWin(p GamePlayer)
	OnGameFinish(r GameRank)
	OnPlayerLoose(g *Game, p GamePlayer)
	OnTargetReached(p GamePlayer, dist float64)
	OnPlayerNearToTarget(p GamePlayer, dist float64)
}

// GameRole represents GamePlayer role
type GameRole string

const (
	// GameRoleUndefined for no role
	GameRoleUndefined GameRole = "undefined"
	// GameRoleTarget for target
	GameRoleTarget GameRole = "target"
	// GameRoleHunter for hunter
	GameRoleHunter GameRole = "hunter"
)

// GamePlayer wraps player and its role in the game
type GamePlayer struct {
	model.Player
	Role GameRole
}

// Game controls rounds and players
type Game struct {
	ID       string
	players  map[string]*GamePlayer
	duration time.Duration
	started  bool
	target   *GamePlayer
	events   GameEvents

	stop context.CancelFunc
}

// NewGame create a game with duration
func NewGame(id string, duration time.Duration, events GameEvents) *Game {
	return &Game{ID: id, events: events, duration: duration, started: false,
		players: make(map[string]*GamePlayer), stop: func() {}}
}

func (g Game) String() string {
	return fmt.Sprintf("%s(%d)started=%v", g.ID, len(g.players), g.started)
}

/*
Start the game
*/
func (g *Game) Start(ctx context.Context) error {
	if g.started {
		return ErrAlreadyStarted
	}

	log.Println("game:", g.ID, ":start!!!!!!")
	g.setPlayersRoles()

	g.started = true

	go g.handleGameFinishEvent(ctx)
	return nil
}

func (g *Game) handleGameFinishEvent(ctx context.Context) {
	var gameCtx context.Context
	gameCtx, g.stop = context.WithTimeout(ctx, g.duration)
	<-gameCtx.Done()
	g.started = false
	g.finish(gameCtx)
}

func (g *Game) finish(ctx context.Context) {
	log.Println("game:", g.ID, ":stop!!!!!!!")
	g.started = false

	_, stillInTheGame := g.players[g.target.ID]
	if stillInTheGame {
		g.events.OnTargetWin(*g.target)
	}

	rank := NewGameRank(g.ID).ByPlayersDistanceToTarget(g.players, *g.target)
	g.events.OnGameFinish(rank)
	g.players = make(map[string]*GamePlayer)
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
	PlayerIDs  []string     `json:"-"`
}

// NewGameRank creates a GameRank
func NewGameRank(gameName string) *GameRank {
	return &GameRank{Game: gameName, PlayerRank: make([]PlayerRank, 0), PlayerIDs: make([]string, 0)}
}

// ByPlayersDistanceToTarget returns a game rank for players based on minimum distance to the target player
func (rank GameRank) ByPlayersDistanceToTarget(players map[string]*GamePlayer, target GamePlayer) GameRank {
	playersDistToTarget := map[int]GamePlayer{}
	for _, p := range players {
		dist := p.DistTo(target.Player)
		playersDistToTarget[int(dist)] = *p
		rank.PlayerIDs = append(rank.PlayerIDs, p.Player.ID)
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

// Started true when game started
func (g Game) Started() bool {
	return g.started
}

/*
SetPlayer notify player updates to the game
The rule is:
    - the game changes what to do with the player
    - it can ignore anything
    - it can send messages to the player
    - it receives sessions to notify anything to this player games
*/
func (g *Game) SetPlayer(id string, lon, lat float64) error {
	if !g.started {
		if _, exists := g.players[id]; !exists {
			log.Printf("game:%s:detect=enter:%s\n", g.ID, id)
			g.players[id] = &GamePlayer{model.Player{ID: id, Lon: lon, Lat: lat}, GameRoleUndefined}
		}
		return nil
	}
	p, exists := g.players[id]
	if !exists {
		return nil
	}
	p.Lon, p.Lat = lon, lat

	if p.Role == GameRoleHunter {
		return g.notifyToTheHunterTheDistanceToTheTarget(p)
	}
	return nil
}

func (g *Game) notifyToTheHunterTheDistanceToTheTarget(p *GamePlayer) error {
	target, exists := g.players[g.target.ID]
	if !exists {
		return ErrPlayerIsNotInTheGame
	}
	dist := p.DistTo(target.Player)

	if dist <= 20 {
		log.Printf("game:%s:detect=winner:%s:dist:%f\n", g.ID, p.ID, dist)
		delete(g.players, target.ID)
		g.events.OnPlayerLoose(g, *target)
		g.events.OnTargetReached(*p, dist)
		g.stop()
	} else if dist <= 100 {
		g.events.OnPlayerNearToTarget(*p, dist)
	}
	return nil
}

/*
RemovePlayer revices notifications to remove player
The role is:
    - it can ignore everthing
    - it receives sessions to send messages to its players
    - it must remove players from the game
*/
func (g *Game) RemovePlayer(id string) {
	gamePlayer, exists := g.players[id]
	if !exists {
		return
	}
	delete(g.players, id)
	if !g.started {
		log.Println("game:"+g.ID+":detect=exit:", gamePlayer)
		return
	}

	if len(g.players) == 1 {
		log.Println("game:"+g.ID+":detect=last-one:", gamePlayer)
		g.stop()
	} else if id == g.target.ID {
		log.Println("game:"+g.ID+":detect=target-loose:", gamePlayer)
		go g.events.OnPlayerLoose(g, *gamePlayer)
		g.stop()
	} else if len(g.players) == 0 {
		log.Println("game:"+g.ID+":detect=no-players:", gamePlayer)
		g.players[id] = gamePlayer
		g.stop()
	} else {
		log.Println("game:"+g.ID+":detect=loose:", gamePlayer)
		go g.events.OnPlayerLoose(g, *gamePlayer)
	}
	return
}

func (g *Game) setPlayersRoles() {
	g.target = sortTargetPlayer(g.players)
	g.target.Role = GameRoleTarget

	for id, p := range g.players {
		if id != g.target.ID {
			p.Role = GameRoleHunter
		}
		g.events.OnGameStarted(g, *p)
	}
}

func sortTargetPlayer(players map[string]*GamePlayer) *GamePlayer {
	ids := make([]string, 0)
	for id := range players {
		ids = append(ids, id)
	}
	return players[ids[rand.Intn(len(ids))]]
}
