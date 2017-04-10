package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/allegro/bigcache"
	io "github.com/googollee/go-socket.io"
)

// MinPlayersPerGame ...
const MinPlayersPerGame = 3

// Game controls rounds and players
type Game struct {
	ID             string
	players        *bigcache.BigCache
	duration       time.Duration
	started        bool
	targetPlayerID string

	stopFunc context.CancelFunc
}

// NewGame create a game with duration
func NewGame(id string, duration time.Duration) *Game {
	cache, _ := bigcache.NewBigCache(bigcache.DefaultConfig(10 * time.Minute))
	return &Game{ID: id, duration: duration, started: false, players: cache}
}

func (g Game) String() string {
	return fmt.Sprintf("%s(%d)started=%v", g.ID, g.players.Len(), g.started)
}

// Start the game
func (g *Game) Start(sessions *SessionManager) {
	if g.started {
		g.Stop()
	}

	log.Println("---------------------------")
	log.Println("Game:", g.ID, ":start!!!!!!")
	log.Println("---------------------------")
	g.sortTargetPlayer()
	g.started = true

	for _, p := range g.playerList() {
		if err := sessions.Emit(p.ID, "game:started", `"`+g.ID+`"`); err != nil {
			log.Println("error to emit game:started", p.ID, err)
		}
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), g.duration)
		g.stopFunc = cancel

		<-ctx.Done()
		log.Println("---------------------------")
		log.Println("Game:", g.ID, ":stop!!!!!!")
		log.Println("---------------------------")
		for _, p := range g.playerList() {
			if err := sessions.Emit(p.ID, "game:finish", `"`+g.ID+`"`); err != nil {
				log.Println("error to emit game:started", p.ID, err)
			}
		}

		g.started = false
		g.players.Reset()
		g.targetPlayerID = ""
	}()
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
	return !g.started && g.players.Len() >= MinPlayersPerGame
}

// WatchPlayers events
func (g *Game) WatchPlayers(stream EventStream, sessions *SessionManager) {
	go stream.StreamIntersects("player", "geofences", g.ID, func(d *Detection) {
		p := &Player{ID: d.FeatID, X: d.Lat, Y: d.Lon}
		switch d.Intersects {
		case Enter:
			g.setPlayerUntilReady(p, sessions)
		case Exit:
			g.removePlayer(p, sessions)
		case Inside:
			if !g.started {
				g.setPlayerUntilReady(p, sessions)
			} else if g.hasPlayer(p.ID) {
				g.setPlayer(p)
				if p.ID != g.targetPlayerID {
					g.updateAndNofityPlayer(p, sessions)
				} else {
					log.Printf("Game:%s:target:move", g.ID)
				}
			}
		}
	})
}

func (g *Game) setPlayerUntilReady(p *Player, sessions *SessionManager) {
	if g.started {
		return
	}
	if !g.hasPlayer(p.ID) {
		log.Println("Game:"+g.ID+":detect=enter:", p)
	}
	g.setPlayer(p)
	if g.Ready() {
		g.Start(sessions)
	}
}

func (g *Game) updateAndNofityPlayer(p *Player, sessions *SessionManager) {
	targetPlayer, err := g.getPlayer(g.targetPlayerID)
	if err != nil || targetPlayer == nil {
		log.Printf("Game:%s:move error:target player missing\n", g.ID)
		g.Stop()
		return
	}

	dist := p.DistTo(targetPlayer)
	if dist <= 20 {
		log.Printf("Game:%s:detect=winner:%s:dist:%f\n", g.ID, p.ID, dist)
		sessions.Emit(p.ID, "target:reached", strconv.FormatFloat(dist, 'f', 0, 64))
		g.Stop()
	} else if dist <= 100 {
		sessions.Emit(p.ID, "target:near", strconv.FormatFloat(dist, 'f', 0, 64))
		log.Printf("Game:%s:detect=near:%s:dist:%f\n", g.ID, p.ID, dist)
	} else {
		log.Printf("Game:%s:detect=far:%s:dist:%f\n", g.ID, p.ID, dist)
	}
}

func (g *Game) setPlayer(p *Player) {
	if p.ID == "" {
		return
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(p); err != nil {
		return
	}
	g.players.Set(p.ID, buf.Bytes())
}

func (g *Game) getPlayer(id string) (*Player, error) {
	data, _ := g.players.Get(id)
	buf := bytes.NewBuffer(data)
	var player Player
	if err := gob.NewDecoder(buf).Decode(&player); err != nil {
		return nil, err
	}
	if player.ID == "" {
		return nil, errors.New("err:getPlayer:player not found:" + id)
	}
	return &player, nil
}

func (g *Game) removePlayer(p *Player, sessions *SessionManager) {
	if !g.hasPlayer(p.ID) {
		return
	}

	g.players.Set(p.ID, nil)
	if !g.started {
		log.Println("Game:"+g.ID+":detect=exit:", p)
		return
	}

	if len(g.playerList()) == 1 {
		log.Println("Game:"+g.ID+":detect=winner:", g.lastPlayer())
		g.Stop()
	} else if p.ID == g.targetPlayerID {
		log.Println("Game:"+g.ID+":detect=target-loose:", p)
		g.Stop()
	} else if g.players.Len() == 0 {
		log.Println("Game:"+g.ID+":detect=no-players:", p)
		g.Stop()
	} else {
		log.Println("Game:"+g.ID+":detect=loose:", p)
		sessions.Emit(p.ID, "game:loose", "{}")
	}
}

func (g *Game) hasPlayer(id string) bool {
	player, err := g.getPlayer(id)
	return err == nil && player != nil
}

func (g *Game) playerList() []*Player {
	players, it := make([]*Player, 0), g.players.Iterator()
	for entry, _ := it.Value(); it.SetNext(); entry, _ = it.Value() {
		log.Println("->", entry.Key())
		if p, _ := g.getPlayer(entry.Key()); p != nil {
			log.Println("->", p)
			players = append(players, p)
		}
	}
	return players
}

func (g *Game) lastPlayer() *Player {
	players := g.playerList()
	if len(players) == 0 {
		return nil
	}
	return players[len(players)-1]
}

func (g *Game) sortTargetPlayer() {
	players := g.playerList()
	randIdx := rand.Intn(len(players))
	g.targetPlayerID = players[randIdx].ID
}

func handleGames(stream EventStream, sessions *SessionManager) {
	games := make(map[string]*Game)
	err := stream.StreamNearByEvents("player", "geofences", 0, func(d *Detection) {
		gameID := d.NearByFeatID
		game, exists := games[gameID]
		if !exists {
			log.Println("Creating game", gameID)
			gameDuration := time.Minute
			game = NewGame(gameID, gameDuration)
			games[gameID] = game
			game.WatchPlayers(stream, sessions)
		}
	})
	if err != nil {
		log.Println("Error to stream geofence:event", err)
	}
}

func handleCheckointsDetection(stream EventStream, sessions *SessionManager, server *io.Server) {
	err := stream.StreamNearByEvents("player", "checkpoint", 1000, func(d *Detection) {
		payload, _ := json.Marshal(d)
		if err := sessions.Emit(d.FeatID, "checkpoint:detected", string(payload)); err != nil {
			log.Println("Error to notify player", d.FeatID, err)
		}
		server.BroadcastTo("main", "admin:feature:checkpoint", d)
	})
	if err != nil {
		log.Println("Error to stream geofence:event", err)
	}
}
