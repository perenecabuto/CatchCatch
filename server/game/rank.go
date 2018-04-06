package game

import (
	"fmt"
	"sort"
)

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

func (r Rank) String() string {
	return fmt.Sprintf("[Game:%s Rank:%+v]", r.Game, r.PlayerRank)
}

// ByPlayersDistanceToTarget returns a game rank for players based on minimum distance to the target player
func (r Rank) ByPlayersDistanceToTarget(players []Player) Rank {
	if len(players) == 0 {
		return r
	}
	playersDistToTarget := map[Player]float64{}
	for _, p := range players {
		playersDistToTarget[p] = p.DistToTarget
		r.PlayerIDs = append(r.PlayerIDs, p.Player.ID)
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
		r.PlayerRank = append(r.PlayerRank, PlayerRank{Player: p.ID, Points: points})
	}

	return r
}
