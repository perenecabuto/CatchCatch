package main

import (
	"fmt"

	"github.com/perenecabuto/CatchCatch/catchcatch-server/model"
	gjson "github.com/tidwall/gjson"
)

// PlayerLocationService manage players and features
type PlayerLocationService interface {
	Register(p *model.Player) error
	Update(p *model.Player) error
	Remove(p *model.Player) error
	Players() (model.PlayerList, error)
}

// Tile38PlayerLocationService manages player locations
type Tile38PlayerLocationService struct {
	repo Repository
}

// NewPlayerLocationService build a PlayerLocationService
func NewPlayerLocationService(repo Repository) PlayerLocationService {
	return &Tile38PlayerLocationService{repo}
}

// Register add new player
func (s *Tile38PlayerLocationService) Exists(p *model.Player) (bool, error) {
	return s.repo.Exists("player", p.ID)
}

// Register add new player
func (s *Tile38PlayerLocationService) Register(p *model.Player) error {
	return s.Update(p)
}

// Update player data
func (s *Tile38PlayerLocationService) Update(p *model.Player) error {
	_, err := s.repo.SetFeature("player", p.ID,
		fmt.Sprintf(`{"type": "Point", "coordinates": [%f, %f]}`, p.Lon, p.Lat))
	return err
}

// Remove player
func (s *Tile38PlayerLocationService) Remove(p *model.Player) error {
	return s.repo.RemoveFeature("player", p.ID)
}

// Players return all registered players
func (s *Tile38PlayerLocationService) Players() (model.PlayerList, error) {
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
