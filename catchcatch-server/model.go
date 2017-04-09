package main

import (
	"encoding/json"
	"errors"
	"fmt"

	redis "gopkg.in/redis.v5"
)

// Feature wraps geofence name and its geojeson
type Feature struct {
	ID          string `json:"id"`
	Group       string `json:"group"`
	Coordinates string `json:"coords"`
}

// Player payload
type Player struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

func (p *Player) String() string {
	return fmt.Sprintln("id:", p.ID, "x:", p.X, "y:", p.Y)
}

// PlayerList payload for list of players
type PlayerList struct {
	Players []*Player `json:"players"`
}

// PlayerLocationService manages player locations
type PlayerLocationService struct {
	client *redis.Client
}

// Register add new player
func (s *PlayerLocationService) Register(p *Player) error {
	return s.Update(p)
}

// Update player data
func (s *PlayerLocationService) Update(p *Player) error {
	cmd := redis.NewStringCmd("SET", "player", p.ID, "POINT", p.X, p.Y)
	s.client.Process(cmd)
	return cmd.Err()
}

// Remove player
func (s *PlayerLocationService) Remove(p *Player) error {
	cmd := redis.NewStringCmd("DEL", "player", p.ID)
	s.client.Process(cmd)
	return cmd.Err()
}

type geom struct {
	Coords [2]float64 `json:"coordinates"`
}

// Players return all registred players
func (s *PlayerLocationService) Players() (*PlayerList, error) {
	features, err := scanFeature(s.client, "player")
	if err != nil {
		return nil, err
	}
	list := &PlayerList{make([]*Player, len(features))}
	for i, f := range features {
		var geo geom
		json.Unmarshal([]byte(f.Coordinates), &geo)
		list.Players[i] = &Player{ID: f.ID, X: geo.Coords[1], Y: geo.Coords[0]}
	}
	return list, nil
}

// PlayerByID search for a player by id
func (s *PlayerLocationService) PlayerByID(id string) (*Player, error) {
	cmd := redis.NewStringCmd("GET", "player", id)
	s.client.Process(cmd)
	data, err := cmd.Result()
	if err != nil {
		return nil, errors.New("PlayerByID: " + err.Error())
	}
	var geo geom
	if err := json.Unmarshal([]byte(data), &geo); err != nil {
		return nil, err
	}
	return &Player{ID: id, X: geo.Coords[1], Y: geo.Coords[0]}, nil
}

// AddFeature persist features
func (s *PlayerLocationService) AddFeature(group, id, geojson string) (*Feature, error) {
	cmd := redis.NewStringCmd("SET", group, id, "OBJECT", geojson)
	s.client.Process(cmd)
	if err := cmd.Err(); err != nil {
		return nil, err
	}
	return &Feature{ID: id, Coordinates: geojson, Group: group}, nil
}

// Features ...
func (s *PlayerLocationService) Features(group string) ([]*Feature, error) {
	return scanFeature(s.client, group)
}

func scanFeature(client *redis.Client, group string) ([]*Feature, error) {
	cmd := redis.NewSliceCmd("SCAN", group)
	client.Process(cmd)
	res, err := cmd.Result()
	if err != nil {
		return nil, err
	}

	payload, _ := redis.NewSliceResult(res[1].([]interface{}), err).Result()
	features := make([]*Feature, len(payload))
	for i, item := range payload {
		itemRes, _ := redis.NewSliceResult(item.([]interface{}), nil).Result()
		features[i] = &Feature{ID: itemRes[0].(string), Coordinates: itemRes[1].(string), Group: group}
	}
	return features, nil
}
