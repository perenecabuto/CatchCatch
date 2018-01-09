package game

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	"github.com/perenecabuto/CatchCatch/server/model"
)

// GameEventName represent game events
type GameEventName string

// GameEventName options
const (
	GameCreated               GameEventName = "game:created"
	GameStarted               GameEventName = "game:started"
	GameFinished              GameEventName = "game:finished"
	GameNothingHappens        GameEventName = "game:nothing"
	GamePlayerAdded           GameEventName = "game:player:added"
	GamePlayerRemoved         GameEventName = "game:player:removed"
	GameTargetWin             GameEventName = "game:target:win"
	GameLastPlayerDetected    GameEventName = "game:finish"
	GamePlayerLoose           GameEventName = "game:player:loose"
	GameTargetLoose           GameEventName = "game:target:reached"
	GamePlayerNearToTarget    GameEventName = "game:player:near"
	GameRunningWithoutPlayers GameEventName = "game:empty"
)

// GameEvent is returned when something happens in the game
type GameEvent struct {
	Name   GameEventName
	Player GamePlayer
}

var (
	// ErrAlreadyStarted happens when an action is denied on running game
	ErrAlreadyStarted = errors.New("game already started")
	// ErrPlayerIsNotInTheGame happens when try to change or remove an player not in the game
	ErrPlayerIsNotInTheGame = errors.New("player is not in this game")
	// GameEventNothing is the NULL event
	GameEventNothing = GameEvent{Name: GameNothingHappens}
)

// GameRole represents GamePlayer role
type GameRole string

// GameRole options
const (
	GameRoleUndefined GameRole = "undefined"
	GameRoleTarget    GameRole = "target"
	GameRoleHunter    GameRole = "hunter"
)

// GamePlayer wraps player and its role in the game
type GamePlayer struct {
	model.Player
	Role         GameRole
	DistToTarget float64
	Loose        bool
}

func (gp GamePlayer) String() string {
	return fmt.Sprintf("[ID: %s, Role: %s, DistToTarget: %f, Loose: %v]",
		gp.ID, gp.Role, gp.DistToTarget, gp.Loose)
}

// Game controls rounds and players
type Game struct {
	ID       string
	started  bool
	players  map[string]*GamePlayer
	targetID string
}

// NewGame create a game with duration
func NewGame(id string) *Game {
	return &Game{ID: id, started: false,
		players: make(map[string]*GamePlayer)}
}

func NewGameWithParams(gameID string, started bool, players map[string]*GamePlayer, targetID string) *Game {
	return &Game{gameID, started, players, targetID}
}

func (g Game) TargetID() string {
	return g.targetID
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
func (rank GameRank) ByPlayersDistanceToTarget(players []GamePlayer) GameRank {
	if len(players) == 0 {
		return rank
	}
	playersDistToTarget := map[GamePlayer]float64{}
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
		if !p.Loose {
			part := float64(dist) / float64(maxDist)
			points = int(100 * part)
		}
		rank.PlayerRank = append(rank.PlayerRank, PlayerRank{Player: p.ID, Points: points})
	}

	return rank
}

// Rank returns the rank of the players in this game
func (g Game) Rank() GameRank {
	players := g.Players()
	return NewGameRank(g.ID).ByPlayersDistanceToTarget(players)
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
			g.players[id] = &GamePlayer{
				model.Player{ID: id, Lon: lon, Lat: lat}, GameRoleUndefined, 0, false}
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
		target := g.players[g.targetID]
		p.DistToTarget = p.DistTo(target.Player)
		if p.DistToTarget <= 20 {
			target.Loose = true
			return GameEvent{Name: GameTargetLoose, Player: *p}, nil
		} else if p.DistToTarget <= 100 {
			return GameEvent{Name: GamePlayerNearToTarget, Player: *p}, nil
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
func (g *Game) RemovePlayer(id string) (GameEvent, error) {
	p, exists := g.players[id]
	if !exists {
		return GameEventNothing, ErrPlayerIsNotInTheGame
	}
	if !g.started {
		delete(g.players, id)
		return GameEvent{Name: GamePlayerRemoved, Player: *p}, nil
	}

	g.players[id].Loose = true
	playersInGame := make([]*GamePlayer, 0)
	for _, gp := range g.players {
		if !gp.Loose {
			playersInGame = append(playersInGame, gp)
		}
	}
	if len(playersInGame) == 1 {
		return GameEvent{Name: GameLastPlayerDetected, Player: *p}, nil
	} else if len(playersInGame) == 0 {
		return GameEvent{Name: GameRunningWithoutPlayers, Player: *p}, nil
	} else if id == g.targetID {
		return GameEvent{Name: GameTargetLoose, Player: *p}, nil
	}

	return GameEvent{Name: GamePlayerLoose, Player: *p}, nil
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

func raffleTargetPlayer(players map[string]*GamePlayer) string {
	rand.New(rand.NewSource(time.Now().Unix()))
	ids := make([]string, 0)
	for id := range players {
		ids = append(ids, id)
	}
	return ids[rand.Intn(len(ids))]
}
