package game

import (
	"fmt"
	"math"
	"sort"

	funk "github.com/thoas/go-funk"
)

// PlayerRank ...
type PlayerRank struct {
	Player Player `json:"player"`
	Points int    `json:"points"`
}

// PlayerRankList ...
type PlayerRankList []PlayerRank

// Rank ...
type Rank struct {
	Game       string         `json:"game"`
	PlayerRank PlayerRankList `json:"points_per_player"`
}

// NewGameRank creates a Rank
func NewGameRank(gameName string) *Rank {
	return &Rank{Game: gameName, PlayerRank: make(PlayerRankList, 0)}
}

func (r Rank) String() string {
	return fmt.Sprintf("[Game:%s Rank:%v]", r.Game, funk.Map(r.PlayerRank, func(pr PlayerRank) string {
		return fmt.Sprintf("\n%s = %d", pr.Player.ID, pr.Points)
	}))
}

/*
ByPlayersDistanceToTarget returns a game rank for players based on minimum distance to the target player

Notice: when lose points it receive ZERO points

formula:
	MD = max(player dist) + 1
	PDT = player dist to target
	POINTS = 100 * (PDT - MD) / MD
*/
func (r Rank) ByPlayersDistanceToTarget(players []Player) Rank {
	if len(players) == 0 {
		return r
	}
	dists := make([]float64, 0)
	for _, p := range players {
		if p.Role == GameRoleTarget {
			dists = append(dists, 0)
		} else {
			dists = append(dists, p.DistToTarget)
		}
	}

	sort.Float64s(dists)
	maxDist := dists[len(dists)-1] + 1

	for _, p := range players {
		var points int
		if p.Lose {
			points = 0
		} else {
			part := float64(maxDist-p.DistToTarget) / float64(maxDist)
			points = int(math.Round(100 * part))
		}
		r.PlayerRank = append(r.PlayerRank, PlayerRank{Player: p, Points: points})
	}

	sort.Sort(r.PlayerRank)
	return r
}

func (pr PlayerRankList) Len() int {
	return len(pr)
}

func (pr PlayerRankList) Less(i int, j int) bool {
	return pr[i].Player.ID < pr[j].Player.ID
}

func (pr PlayerRankList) Swap(i int, j int) {
	pr[i], pr[j] = pr[j], pr[i]
}
