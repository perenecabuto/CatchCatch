package main

import (
	"encoding/json"
	"fmt"
	"log"

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
	conn redis.Conn
}

func (s *PlayerLocationService) Register(p *Player) error {
	return s.conn.Send("SET", "player", p.ID, "POINT", p.X, p.Y)
}

func (s *PlayerLocationService) Update(p *Player) error {
	return s.conn.Send("SET", "player", p.ID, "POINT", p.X, p.Y)
}

func (s *PlayerLocationService) Remove(p *Player) error {
	return s.conn.Send("DEL", "player", p.ID)
}

func (s *PlayerLocationService) All() (*PlayerList, error) {
	result, err := s.conn.Do("SCAN", "player")
	if err != nil {
		return nil, err
	}

	var payload []interface{}
	redis.Scan(result.([]interface{}), nil, &payload)

	list := make([]*Player, len(payload))
	for i, d := range payload {
		var id string
		var data []byte
		redis.Scan(d.([]interface{}), &id, &data)
		geo := &struct {
			Coords [2]float32 `json:"coordinates"`
		}{}
		json.Unmarshal(data, geo)
		list[i] = &Player{ID: id, X: geo.Coords[0], Y: geo.Coords[1]}
	}
	return &PlayerList{list}, nil
}

func (s *PlayerLocationService) GetPlayer(id string) (*Player, error) {
	result, err := s.conn.Do("GET", "player", id)
	if err != nil {
		return nil, err
	}

	log.Println("->", string(result.([]uint8)))
	return &Player{}, nil
}
