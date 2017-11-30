package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"

	"github.com/perenecabuto/CatchCatch/catchcatch-server/model"
)

type GameEventName string

var (
	// ErrAlreadyStarted happens when an action is denied on running game
	ErrAlreadyStarted = errors.New("game already started")
	// ErrPlayerIsNotInTheGame happens when try to change or remove an player not in the game
	ErrPlayerIsNotInTheGame = errors.New("player is not in this game")

	GameCreated               GameEventName = "game:created"
	GameStarted               GameEventName = "game:started"
	GameFinished              GameEventName = "game:finished"
	GameNothingHappens        GameEventName = "game:nothing"
	GamePlayerAdded           GameEventName = "game:player:added"
	GamePlayerRemoved         GameEventName = "game:player:removed"
	GameTargetWin             GameEventName = "game:target:win"
	GameLastPlayerDetected    GameEventName = "game:finish"
	GamePlayerLoose           GameEventName = "game:player:loose"
	GameTargetReached         GameEventName = "game:target:reached"
	GamePlayerNearToTarget    GameEventName = "game:player:near"
	GameRunningWithoutPlayers GameEventName = "game:empty"

	GameEventNothing = GameEvent{Name: GameNothingHappens}
)

type GameEvent struct {
	Name   GameEventName
	Player GamePlayer
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
	Role         GameRole
	DistToTarget float64
}

// Game controls rounds and players
type Game struct {
	ID      string
	players map[string]*GamePlayer
	started bool
	target  *GamePlayer
}

// NewGame create a game with duration
func NewGame(id string) *Game {
	return &Game{ID: id, started: false,
		players: make(map[string]*GamePlayer)}
}

func (g Game) String() string {
	return fmt.Sprintf("%s(%d)started=%v", g.ID, len(g.players), g.started)
}

/*
Start the game
*/
func (g *Game) Start() GameEvent {
	if g.started {
		return GameEventNothing
	}
	log.Println("game:", g.ID, ":start!!!!!!")
	g.setPlayersRoles()
	g.started = true
	return GameEvent{Name: GameStarted}
}

// Stop the game
func (g *Game) Stop() GameEvent {
	if !g.started {
		return GameEventNothing
	}
	log.Println("game:", g.ID, ":stop!!!!!!!")
	g.started = false
	g.players = make(map[string]*GamePlayer)
	return GameEvent{Name: GameFinished}
}

// Players return game players
func (g *Game) Players() []GamePlayer {
	players := make([]GamePlayer, len(g.players))
	i := 0
	for _, p := range g.players {
		players[i] = *p
		i++
	}
	return players
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
		dist := int(p.DistToTarget)
		playersDistToTarget[dist] = *p
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
func (g *Game) SetPlayer(id string, lon, lat float64) (GameEvent, error) {
	if !g.started {
		if _, exists := g.players[id]; !exists {
			log.Printf("game:%s:detect=enter:%s\n", g.ID, id)
			g.players[id] = &GamePlayer{model.Player{ID: id, Lon: lon, Lat: lat}, GameRoleUndefined, 0}
			return GameEvent{Name: GamePlayerAdded}, nil
		}
		return GameEventNothing, nil
	}
	p, exists := g.players[id]
	if !exists {
		return GameEventNothing, nil
	}
	p.Lon, p.Lat = lon, lat

	if p.Role == GameRoleHunter {
		return g.notifyToTheHunterTheDistanceToTheTarget(p)
	}
	return GameEventNothing, nil
}

func (g *Game) notifyToTheHunterTheDistanceToTheTarget(p *GamePlayer) (GameEvent, error) {
	target, exists := g.players[g.target.ID]
	if !exists {
		return GameEventNothing, ErrPlayerIsNotInTheGame
	}
	p.DistToTarget = p.DistTo(target.Player)

	if p.DistToTarget <= 20 {
		log.Printf("game:%s:detect=winner:%s:dist:%f\n", g.ID, p.ID, p.DistToTarget)
		delete(g.players, target.ID)
		// TODO: remember to notify target that he loses
		// g.events.OnPlayerLoose(*g, *target)
		return GameEvent{Name: GameTargetReached, Player: *p}, nil
	} else if p.DistToTarget <= 100 {
		return GameEvent{Name: GamePlayerNearToTarget, Player: *p}, nil
	}
	return GameEventNothing, nil
}

/*
RemovePlayer revices notifications to remove player
The role is:
    - it can ignore everthing
    - it receives sessions to send messages to its players
    - it must remove players from the game
*/
func (g *Game) RemovePlayer(id string) (GameEvent, error) {
	gamePlayer, exists := g.players[id]
	if !exists {
		return GameEventNothing, ErrPlayerIsNotInTheGame
	}
	delete(g.players, id)
	if !g.started {
		log.Println("game:"+g.ID+":detect=exit:", gamePlayer)
		return GameEvent{Name: GamePlayerRemoved, Player: *gamePlayer}, nil
	}

	if len(g.players) == 1 {
		log.Println("game:"+g.ID+":detect=last-one:", gamePlayer)
		g.Stop()
	} else if id == g.target.ID {
		log.Println("game:"+g.ID+":detect=target-loose:", gamePlayer)
		// TODO: Remove stop
		g.Stop()
		return GameEvent{Name: GamePlayerRemoved, Player: *gamePlayer}, nil
	} else if len(g.players) == 0 {
		log.Println("game:"+g.ID+":detect=no-players:", gamePlayer)
		g.players[id] = gamePlayer
		// TODO: Remove stop
		g.Stop()
	} else {
		log.Println("game:"+g.ID+":detect=loose:", gamePlayer)
		return GameEvent{Name: GamePlayerLoose, Player: *gamePlayer}, nil
	}
	return GameEventNothing, nil
}

func (g *Game) setPlayersRoles() {
	g.target = sortTargetPlayer(g.players)
	g.target.Role = GameRoleTarget

	for id, p := range g.players {
		if id != g.target.ID {
			p.Role = GameRoleHunter
		}
	}
}

func sortTargetPlayer(players map[string]*GamePlayer) *GamePlayer {
	ids := make([]string, 0)
	for id := range players {
		ids = append(ids, id)
	}
	return players[ids[rand.Intn(len(ids))]]
}
