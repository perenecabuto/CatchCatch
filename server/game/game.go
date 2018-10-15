package game

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/perenecabuto/CatchCatch/server/model"
)

// EventName represent game events
type EventName string

// EventName options
const (
	GameStarted               EventName = "game:started"
	GameNothingHappens        EventName = "game:nothing"
	GamePlayerAdded           EventName = "game:player:added"
	GamePlayerRemoved         EventName = "game:player:removed"
	GameLastPlayerDetected    EventName = "game:player:last"
	GamePlayerRanWay          EventName = "game:player:ranaway"
	GameTargetReached         EventName = "game:target:reached"
	GamePlayerNearToTarget    EventName = "game:player:near"
	GameRunningWithoutPlayers EventName = "game:empty"
)

// DefaultMinDistToTarget is the minimum dist from the hunter to target to notify the hunter
const DefaultMinDistToTarget = 100

// Event is returned when something happens in the game
type Event struct {
	Name   EventName
	Player Player
}

var (
	// ErrAlreadyStarted happens when an action is denied on running game
	ErrAlreadyStarted = errors.New("game already started")
	// GameEventNothing is the NULL event
	GameEventNothing = Event{Name: GameNothingHappens}
)

// Role represents Player role
type Role string

// Role options
const (
	GameRoleUndefined Role = "undefined"
	GameRoleTarget    Role = "target"
	GameRoleHunter    Role = "hunter"
)

// Player wraps model.Player and its role in the game
type Player struct {
	model.Player
	Role         Role
	DistToTarget float64
	Lose         bool
}

func (gp Player) String() string {
	return fmt.Sprintf("[ID: %s, Role: %s, DistToTarget: %f, Lose: %v]",
		gp.ID, gp.Role, gp.DistToTarget, gp.Lose)
}

// Game controls rounds and players
type Game struct {
	ID       string
	started  int32
	players  map[string]*Player
	targetID atomic.Value

	playersLock sync.RWMutex
}

// NewGame create a game with duration
func NewGame(id string) *Game {
	var tid atomic.Value
	tid.Store("")
	return &Game{ID: id, started: 0, players: make(map[string]*Player), targetID: tid}
}

// NewGameWithParams ...
func NewGameWithParams(gameID string, started bool, players []Player, targetID string) *Game {
	mPlayers := map[string]*Player{}
	for _, p := range players {
		copy := p
		mPlayers[p.ID] = &copy
	}
	var s int32
	if started {
		s = 1
	}
	var tid atomic.Value
	tid.Store(targetID)
	return &Game{ID: gameID, started: s, players: mPlayers, targetID: tid}
}

// TargetID returns the targe player id
func (g *Game) TargetID() string {
	return g.targetID.Load().(string)
}

func (g *Game) String() string {
	return fmt.Sprintf("[ID:%s|Started:%v|TargetID:%s|Players: %+v]",
		g.ID, g.Started(), g.TargetID(), g.Players())
}

// Start the game
func (g *Game) Start() {
	log.Println("game:", g.ID, ":start!!!!!!")
	g.setPlayersRoles()
	atomic.StoreInt32(&g.started, 1)
}

// Stop the game
func (g *Game) Stop() {
	atomic.StoreInt32(&g.started, 0)
	g.targetID.Store("")
	g.playersLock.Lock()
	g.players = make(map[string]*Player)
	g.playersLock.Unlock()
}

// Players return game players
func (g *Game) Players() []Player {
	var i int
	g.playersLock.Lock()
	players := make([]Player, len(g.players))
	for _, p := range g.players {
		players[i] = *p
		i++
	}
	g.playersLock.Unlock()
	return players
}

// TargetPlayer returns the target player when it's set
func (g *Game) TargetPlayer() *Player {
	g.playersLock.RLock()
	target := g.players[g.TargetID()]
	g.playersLock.RUnlock()
	return target
}

// Info ...
type Info struct {
	Role string `json:"role"`
	Game string `json:"game"`
}

// Rank returns the rank of the players in this game
func (g *Game) Rank() Rank {
	players := g.Players()
	return NewGameRank(g.ID).ByPlayersDistanceToTarget(players)
}

// Started true when game started
func (g *Game) Started() bool {
	return atomic.LoadInt32(&g.started) == 1
}

/*
SetPlayer notify player updates to the game
The rule is:
    - the game changes what to do with the player
    - it can ignore anything
    - it can send messages to the player
    - it receives sessions to notify anything to this player games
*/
func (g *Game) SetPlayer(id string, lon, lat float64) Event {
	g.playersLock.RLock()
	p, exists := g.players[id]
	g.playersLock.RUnlock()

	if exists {
		p.Lat, p.Lon = lat, lon
	} else if !g.Started() {
		log.Printf("game:%s:detect=enter:%s\n", g.ID, id)
		g.playersLock.Lock()
		g.players[id] = &Player{
			model.Player{ID: id, Lon: lon, Lat: lat}, GameRoleUndefined, 0, false}
		g.playersLock.Unlock()
		return Event{Name: GamePlayerAdded}
	} else {
		return GameEventNothing
	}

	if p.Role == GameRoleHunter {
		target := g.TargetPlayer()
		p.DistToTarget = p.DistTo(target.Player)
		if p.DistToTarget <= 20 {
			target.Lose = true
			return Event{Name: GameTargetReached, Player: *p}
		} else if p.DistToTarget <= DefaultMinDistToTarget {
			return Event{Name: GamePlayerNearToTarget, Player: *p}
		}
	}
	return GameEventNothing
}

/*
RemovePlayer revices notifications to remove player
The role is:
    - it can ignore everthing
    - it receives sessions to send messages to its players
    - it must remove players from the game
*/
func (g *Game) RemovePlayer(id string) Event {
	p, exists := g.players[id]
	if !exists {
		return GameEventNothing
	}
	if !g.Started() {
		delete(g.players, id)
		return Event{Name: GamePlayerRemoved, Player: *p}
	}

	g.playersLock.Lock()
	g.players[id].Lose = true
	playersInGame := make([]*Player, 0)
	for _, gp := range g.players {
		if !gp.Lose {
			playersInGame = append(playersInGame, gp)
		}
	}
	g.playersLock.Unlock()

	switch len(playersInGame) {
	case 1:
		return Event{Name: GameLastPlayerDetected, Player: *playersInGame[0]}
	case 0:
		return Event{Name: GameRunningWithoutPlayers}
	default:
		return Event{Name: GamePlayerRanWay, Player: *p}
	}
}

func (g *Game) setPlayersRoles() {
	g.targetID.Store(raffleTargetPlayer(g.players))
	for id, p := range g.players {
		if id == g.TargetID() {
			p.Role = GameRoleTarget
		} else {
			p.DistToTarget = p.DistTo(g.TargetPlayer().Player)
			p.Role = GameRoleHunter
		}
	}
}

func raffleTargetPlayer(players map[string]*Player) string {
	rand.New(rand.NewSource(time.Now().Unix()))
	ids := make([]string, 0)
	for id := range players {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return ""
	}
	return ids[rand.Intn(len(ids))]
}
