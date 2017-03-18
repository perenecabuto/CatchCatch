package main

import (
	"encoding/json"
	"fmt"

	redis "gopkg.in/redis.v5"
)

// Player payload
type Player struct {
	ID string  `json:"id"`
	X  float32 `json:"x"`
	Y  float32 `json:"y"`
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

func (s *PlayerLocationService) Register(p *Player) error {
	return s.Update(p)
}

func (s *PlayerLocationService) Update(p *Player) error {
	cmd := redis.NewStringCmd("SET", "player", p.ID, "POINT", p.X, p.Y)
	s.client.Process(cmd)
	return cmd.Err()
}

func (s *PlayerLocationService) Remove(p *Player) error {
	cmd := redis.NewStringCmd("DEL", "player", p.ID)
	s.client.Process(cmd)
	return cmd.Err()
}

type position struct {
	Coords [2]float32 `json:"coordinates"`
}

func (s *PlayerLocationService) All() (*PlayerList, error) {
	cmd := redis.NewSliceCmd("SCAN", "player")
	s.client.Process(cmd)
	res, err := cmd.Result()
	if err != nil {
		return nil, err
	}

	payload, _ := redis.NewSliceResult(res[1].([]interface{}), err).Result()
	list := make([]*Player, len(payload))
	for i, item := range payload {
		itemRes, _ := redis.NewSliceResult(item.([]interface{}), nil).Result()
		id, data, geo := itemRes[0].(string), []byte(itemRes[1].(string)), &position{}
		json.Unmarshal(data, geo)

		list[i] = &Player{ID: id, X: geo.Coords[0], Y: geo.Coords[1]}
	}

	return &PlayerList{list}, nil
}
