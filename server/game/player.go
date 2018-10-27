package game

import (
	"fmt"
	"sync"

	"github.com/perenecabuto/CatchCatch/server/model"
)

// Player wraps model.Player and its role in the game
type Player struct {
	model.Player
	Role         Role
	DistToTarget float64
	Lose         bool
}

func (p Player) String() string {
	return fmt.Sprintf("[ID: %s, Role: %s, DistToTarget: %f, Lose: %v]",
		p.ID, p.Role, p.DistToTarget, p.Lose)
}

type GamePlayers struct {
	sync.RWMutex
	players map[string]*Player
}

func NewGamePlayers() *GamePlayers {
	return &GamePlayers{players: make(map[string]*Player)}
}

func (gp *GamePlayers) GetByID(id string) (*Player, bool) {
	gp.RLock()
	p, ok := gp.players[id]
	gp.RUnlock()
	return p, ok
}

func (gp *GamePlayers) Set(players ...Player) {
	gp.Lock()
	for _, p := range players {
		copy := p
		gp.players[p.ID] = &copy
	}
	gp.Unlock()
}

func (gp *GamePlayers) SetLoserByID(id string) (*Player, bool) {
	gp.Lock()
	p, ok := gp.players[id]
	if ok {
		gp.players[id].Lose = true
	}
	gp.Unlock()
	return p, ok
}

func (gp *GamePlayers) DeleteByID(id string) {
	gp.RLock()
	delete(gp.players, id)
	gp.RUnlock()
}

func (gp *GamePlayers) DeleteAll() {
	gp.Lock()
	gp.players = make(map[string]*Player)
	gp.Unlock()
}

func (gp *GamePlayers) All() []*Player {
	gp.RLock()
	players := make([]*Player, len(gp.players))
	var i int
	for _, p := range gp.players {
		players[i] = p
		i++
	}
	gp.RUnlock()
	return players
}

func (gp *GamePlayers) Copy() []Player {
	gp.Lock()
	i := 0
	copy := make([]Player, len(gp.players))
	for _, p := range gp.players {
		copy[i] = *p
		i++
	}
	gp.Unlock()
	return copy
}

func (gp *GamePlayers) AllExceptLosers() []*Player {
	gp.Lock()
	players := make([]*Player, 0)
	for _, p := range gp.players {
		if !p.Lose {
			players = append(players, p)
		}
	}
	gp.Unlock()
	return players
}

func (gp *GamePlayers) AsMap() map[string]Player {
	gp.RLock()
	players := make(map[string]Player)
	for id, p := range gp.players {
		players[id] = *p
	}
	gp.RUnlock()
	return players
}
