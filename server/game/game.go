package game

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"
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
	GameTargetWin             EventName = "game:target:win"
	GameLastPlayerDetected    EventName = "game:player:last"
	GamePlayerLose            EventName = "game:player:loose"
	GameTargetLose            EventName = "game:target:reached"
	GamePlayerNearToTarget    EventName = "game:player:near"
	GameRunningWithoutPlayers EventName = "game:empty"
)

// Event is returned when something happens in the game
type Event struct {
	Name   EventName
	Player Player
}

var (
	// ErrAlreadyStarted happens when an action is denied on running game
	ErrAlreadyStarted = errors.New("game already started")
	// ErrPlayerIsNotInTheGame happens when try to change or remove an player not in the game
	ErrPlayerIsNotInTheGame = errors.New("player is not in this game")
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
	started  bool
	players  map[string]*Player
	targetID string

	playersLock sync.RWMutex
}

// NewGame create a game with duration
func NewGame(id string) *Game {
	return &Game{ID: id, started: false, players: make(map[string]*Player)}
}

// NewGameWithParams ...
func NewGameWithParams(gameID string, started bool, players []Player, targetID string) *Game {
	mPlayers := map[string]*Player{}
	for _, p := range players {
		copy := p
		mPlayers[p.ID] = &copy
	}
	return &Game{ID: gameID, started: started, players: mPlayers, targetID: targetID}
}

// TargetID returns the targe player id
func (g *Game) TargetID() string {
	return g.targetID
}

func (g *Game) String() string {
	return fmt.Sprintf("[ID: %s, Started: %v, Players: %+v]",
		g.ID, g.started, g.Players())
}

/*
Start the game
*/
func (g *Game) Start() {
	log.Println("game:", g.ID, ":start!!!!!!")
	g.setPlayersRoles()
	g.started = true
}

// Stop the game
func (g *Game) Stop() {
	g.playersLock.Lock()
	log.Println("game:", g.ID, ":stop!!!!!!!")
	g.started = false
	g.players = make(map[string]*Player)
	g.playersLock.Unlock()
}

// Players return game players
func (g *Game) Players() []Player {
	g.playersLock.Lock()
	players := make([]Player, len(g.players))
	var i int
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
	target := g.players[g.targetID]
	g.playersLock.RUnlock()
	return target
}

// Info ...
type Info struct {
	Role string `json:"role"`
	Game string `json:"game"`
}

// PlayerRank ...
type PlayerRank struct {
	Player string `json:"player"`
	Points int    `json:"points"`
}

// Rank ...
type Rank struct {
	Game       string       `json:"game"`
	PlayerRank []PlayerRank `json:"points_per_player"`
	PlayerIDs  []string     `json:"-"`
}

// NewGameRank creates a Rank
func NewGameRank(gameName string) *Rank {
	return &Rank{Game: gameName, PlayerRank: make([]PlayerRank, 0), PlayerIDs: make([]string, 0)}
}

// ByPlayersDistanceToTarget returns a game rank for players based on minimum distance to the target player
func (rank Rank) ByPlayersDistanceToTarget(players []Player) Rank {
	if len(players) == 0 {
		return rank
	}
	playersDistToTarget := map[Player]float64{}
	for _, p := range players {
		playersDistToTarget[p] = p.DistToTarget
		rank.PlayerIDs = append(rank.PlayerIDs, p.Player.ID)
	}
	dists := make([]float64, 0)
	for _, dist := range playersDistToTarget {
		dists = append(dists, dist)
	}
	sort.Float64s(dists)
	maxDist := dists[len(dists)-1] + 1

	for p, dist := range playersDistToTarget {
		points := 0
		if !p.Lose {
			part := float64(dist) / float64(maxDist)
			points = int(100 * part)
		}
		rank.PlayerRank = append(rank.PlayerRank, PlayerRank{Player: p.ID, Points: points})
	}

	return rank
}

// Rank returns the rank of the players in this game
func (g *Game) Rank() Rank {
	players := g.Players()
	return NewGameRank(g.ID).ByPlayersDistanceToTarget(players)
}

// Started true when game started
func (g *Game) Started() bool {
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
func (g *Game) SetPlayer(id string, lon, lat float64) (Event, error) {
	g.playersLock.RLock()
	p, exists := g.players[id]
	g.playersLock.RUnlock()

	if !g.started {
		if !exists {
			log.Printf("game:%s:detect=enter:%s\n", g.ID, id)

			g.playersLock.Lock()
			g.players[id] = &Player{
				model.Player{ID: id, Lon: lon, Lat: lat}, GameRoleUndefined, 0, false}
			g.playersLock.Unlock()

			return Event{Name: GamePlayerAdded}, nil
		}
		return GameEventNothing, nil
	}
	if !exists {
		return GameEventNothing, nil
	}
	p.Lon, p.Lat = lon, lat

	if p.Role == GameRoleHunter {
		target := g.players[g.targetID]
		p.DistToTarget = p.DistTo(target.Player)
		if p.DistToTarget <= 20 {
			target.Lose = true
			return Event{Name: GameTargetLose, Player: *p}, nil
		} else if p.DistToTarget <= 100 {
			return Event{Name: GamePlayerNearToTarget, Player: *p}, nil
		}
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
func (g *Game) RemovePlayer(id string) (Event, error) {
	p, exists := g.players[id]
	if !exists {
		return GameEventNothing, ErrPlayerIsNotInTheGame
	}
	if !g.started {
		delete(g.players, id)
		return Event{Name: GamePlayerRemoved, Player: *p}, nil
	}

	g.players[id].Lose = true
	playersInGame := make([]*Player, 0)
	for _, gp := range g.players {
		if !gp.Lose {
			playersInGame = append(playersInGame, gp)
		}
	}
	if len(playersInGame) == 1 {
		return Event{Name: GameLastPlayerDetected, Player: *p}, nil
	} else if len(playersInGame) == 0 {
		return Event{Name: GameRunningWithoutPlayers, Player: *p}, nil
	} else if id == g.targetID {
		return Event{Name: GameTargetLose, Player: *p}, nil
	}

	return Event{Name: GamePlayerLose, Player: *p}, nil
}

func (g *Game) setPlayersRoles() {
	g.targetID = raffleTargetPlayer(g.players)
	for id, p := range g.players {
		if id == g.targetID {
			p.Role = GameRoleTarget
		} else {
			p.DistToTarget = p.DistTo(g.players[g.targetID].Player)
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
