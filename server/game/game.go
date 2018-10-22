package game

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
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

// Game controls rounds and players
type Game struct {
	ID       string
	started  int32
	players  *GamePlayers
	targetID atomic.Value
}

// NewGame create a game with duration
func NewGame(id string) *Game {
	var tid atomic.Value
	tid.Store("")
	return &Game{ID: id, started: 0, players: NewGamePlayers(), targetID: tid}
}

// NewGameWithParams ...
func NewGameWithParams(gameID string, started bool, players []Player, targetID string) *Game {
	var s int32
	if started {
		s = 1
	}
	var tid atomic.Value
	tid.Store(targetID)
	mPlayers := NewGamePlayers()
	mPlayers.Set(players...)
	return &Game{ID: gameID, started: s, players: mPlayers, targetID: tid}
}

// TargetID returns the targe player id
func (g *Game) TargetID() string {
	return g.targetID.Load().(string)
}

func (g *Game) String() string {
	return fmt.Sprintf("[ID:%s|Started:%v|TargetID:%s|Players: %+v]",
		g.ID, g.Started(), g.TargetID(), g.players.Copy())
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
	g.players.DeleteAll()
}

// Players return game players
func (g *Game) Players() []Player {
	return g.players.Copy()
}

// TargetPlayer returns the target player when it's set
func (g *Game) TargetPlayer() *Player {
	p, _ := g.players.GetByID(g.TargetID())
	return p
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
	p, exists := g.players.GetByID(id)
	if exists {
		p.Lat, p.Lon = lat, lon
	} else if !g.Started() {
		log.Printf("game:%s:detect=enter:%s\n", g.ID, id)
		g.players.Set(Player{
			model.Player{ID: id, Lon: lon, Lat: lat}, GameRoleUndefined, 0, false})
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
	p, exists := g.players.GetByID(id)
	if !exists {
		return GameEventNothing
	}
	if !g.Started() {
		g.players.DeleteByID(id)
		return Event{Name: GamePlayerRemoved, Player: *p}
	}

	g.players.SetLoserByID(id)
	playersInGame := g.players.AllExceptLosers()

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
	players := g.players.All()
	targetID := raffleTargetPlayer(players)
	g.targetID.Store(targetID)
	for _, p := range players {
		if p.ID == targetID {
			p.Role = GameRoleTarget
		} else {
			p.DistToTarget = p.DistTo(g.TargetPlayer().Player)
			p.Role = GameRoleHunter
		}
	}
}

func raffleTargetPlayer(players []*Player) string {
	rand.New(rand.NewSource(time.Now().Unix()))
	ids := make([]string, 0)
	for _, p := range players {
		ids = append(ids, p.ID)
	}
	if len(ids) == 0 {
		return ""
	}
	return ids[rand.Intn(len(ids))]
}
