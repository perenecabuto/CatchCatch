package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/perenecabuto/CatchCatch/catchcatch-server/model"
	gjson "github.com/tidwall/gjson"
	sjson "github.com/tidwall/sjson"
)

// PlayerLocationService manage players and features
type PlayerLocationService interface {
	Set(p *model.Player) error
	Remove(p *model.Player) error
	All() (model.PlayerList, error)
}

// Tile38PlayerLocationService manages player locations
type Tile38PlayerLocationService struct {
	repo Repository
}

// NewPlayerLocationService build a PlayerLocationService
func NewPlayerLocationService(repo Repository) PlayerLocationService {
	return &Tile38PlayerLocationService{repo}
}

// Exists add new player
func (s *Tile38PlayerLocationService) Exists(p *model.Player) (bool, error) {
	return s.repo.Exists("player", p.ID)
}

// Set player data
func (s *Tile38PlayerLocationService) Set(p *model.Player) error {
	_, err := s.repo.SetFeature("player", p.ID,
		fmt.Sprintf(`{"type": "Point", "coordinates": [%f, %f]}`, p.Lon, p.Lat))
	return err
}

// Remove player
func (s *Tile38PlayerLocationService) Remove(p *model.Player) error {
	return s.repo.RemoveFeature("player", p.ID)
}

// All return all registered players
func (s *Tile38PlayerLocationService) All() (model.PlayerList, error) {
	features, err := s.repo.Features("player")
	if err != nil {
		return nil, err
	}
	list := make(model.PlayerList, len(features))
	for i, f := range features {
		coords := gjson.Get(f.Coordinates, "coordinates").Array()
		if len(coords) != 2 {
			coords = make([]gjson.Result, 2)
		}
		list[i] = &model.Player{ID: f.ID, Lat: coords[1].Float(), Lon: coords[0].Float()}
	}
	return list, nil
}

type GameService interface {
	Create(gameID, serverID string) error
	Update(g *Game, serverID string, evt GameEvent) error
	Remove(gameID string) error
	GameByID(gameID string) (*Game, *GameEvent, error)
}

type Tile38GameService struct {
	repo   Repository
	stream EventStream
}

func NewGameService(repo Repository, stream EventStream) GameService {
	return &Tile38GameService{repo, stream}
}

func (gs *Tile38GameService) Create(gameID string, serverID string) error {
	f, err := gs.repo.FeatureByID("geofences", gameID)
	if err != nil {
		return err
	}
	// TODO: set the server name
	// TODO: remove game when it finishes
	_, err = gs.repo.SetFeature("game", gameID, f.Coordinates)
	if err != nil {
		return err
	}

	extra, _ := sjson.Set("", "updated_at", time.Now().Unix())
	extra, _ = sjson.Set(extra, "server_id", serverID)
	return gs.repo.SetFeatureExtraData("game", gameID, extra)
}

func (gs *Tile38GameService) Update(g *Game, serverID string, evt GameEvent) error {
	extra, _ := sjson.Set("", "event", evt)
	extra, _ = sjson.Set(extra, "updated_at", time.Now().Unix())
	extra, _ = sjson.Set(extra, "server_id", serverID)
	extra, _ = sjson.Set(extra, "players", g.Players())
	extra, _ = sjson.Set(extra, "started", g.Started())
	return gs.repo.SetFeatureExtraData("game", g.ID, extra)
}

func (gs *Tile38GameService) GameByID(gameID string) (*Game, *GameEvent, error) {
	data, err := gs.repo.FeatureExtraData("game", gameID)
	if err == ErrFeatureNotFound {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	players := make(map[string]*GamePlayer)
	pdata := gjson.Get(data, "players").Array()
	for _, pd := range pdata {
		p := &GamePlayer{
			Player:       model.Player{ID: pd.Get("id").String(), Lon: pd.Get("lon").Float(), Lat: pd.Get("lat").Float()},
			Role:         GameRole(pd.Get("Role").String()),
			DistToTarget: pd.Get("DistToTarget").Float(),
			Loose:        pd.Get("Loose").Bool(),
		}
		players[p.ID] = p
	}

	edata := gjson.Get(data, "event")
	evt := &GameEvent{Name: GameEventName(edata.Get("Name").String())}
	if p := players[edata.Get("Player.ID").String()]; p != nil {
		evt.Player = *p
	}

	started := gjson.Get(data, "started").Bool()
	var target *GamePlayer
	for _, p := range players {
		if p.Role == GameRoleTarget {
			target = p
			break
		}
	}
	return &Game{gameID, started, players, target}, evt, nil
}

func (gs *Tile38GameService) Remove(gameID string) error {
	if err := gs.repo.DelFeatureExtraData("game", gameID); err != nil {
		return err
	}
	return gs.repo.RemoveFeature("game", gameID)
}
