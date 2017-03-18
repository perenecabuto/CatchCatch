package main

import (
	"encoding/json"
	"fmt"

	"github.com/garyburd/redigo/redis"
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
	pool *redis.Pool
}

func (s *PlayerLocationService) Register(p *Player) error {
	conn, err := s.pool.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Send("SET", "player", p.ID, "POINT", p.X, p.Y)
}

func (s *PlayerLocationService) Update(p *Player) error {
	conn, err := s.pool.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Send("SET", "player", p.ID, "POINT", p.X, p.Y)
}

func (s *PlayerLocationService) Remove(p *Player) error {
	conn, err := s.pool.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Send("DEL", "player", p.ID)
}

type geo struct {
	Coords [2]float32 `json:"coordinates"`
}

func (s *PlayerLocationService) All() (*PlayerList, error) {
	conn, err := s.pool.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var payload []interface{}
	result, err := conn.Do("SCAN", "player")
	redis.Scan(result.([]interface{}), nil, &payload)

	list := make([]*Player, len(payload))
	for i, d := range payload {
		var id string
		var data []byte
		redis.Scan(d.([]interface{}), &id, &data)
		var geo *geo
		json.Unmarshal(data, geo)
		list[i] = &Player{ID: id, X: geo.Coords[0], Y: geo.Coords[1]}
	}
	return &PlayerList{list}, nil
}
