package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	redis "gopkg.in/redis.v5"
)

// Feature wraps geofence name and its geojeson
type Feature struct {
	ID          string `json:"id"`
	Coordinates string `json:"coords"`
}

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

// Players return all registred players
func (s *PlayerLocationService) Players() (*PlayerList, error) {
	features, err := scanFeature(s.client, "player")
	if err != nil {
		return nil, err
	}
	list := &PlayerList{make([]*Player, len(features))}
	for i, f := range features {
		var geo struct {
			Coords [2]float32 `json:"coordinates"`
		}
		json.Unmarshal([]byte(f.Coordinates), &geo)
		list.Players[i] = &Player{ID: f.ID, X: geo.Coords[1], Y: geo.Coords[0]}
	}
	return list, nil
}

// AddGeofence persist geofence
func (s *PlayerLocationService) AddGeofence(name string, geojson string) error {
	cmd := redis.NewStringCmd("SET", "mapfences", name, "OBJECT", geojson)
	s.client.Process(cmd)
	return cmd.Err()
}

// Geofences ...
func (s *PlayerLocationService) Geofences() ([]*Feature, error) {
	return scanFeature(s.client, "mapfences")
}

// StreamGeofenceEvents ...
func (s *PlayerLocationService) StreamGeofenceEvents(addr string, callback func(msg string)) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	cmd := "NEARBY player FENCE ROAM mapfences * 0\r\n"
	log.Println("REDIS DEBUG:", cmd)
	if _, err = fmt.Fprintf(conn, cmd); err != nil {
		return err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	res := string(buf[:n])
	if res != "+OK\r\n" {
		return fmt.Errorf("expected OK, got '%v'", res)
	}

	t := time.NewTicker(100 * time.Microsecond)
	for range t.C {
		if n, err = conn.Read(buf); err != nil {
			return err
		}
		for _, line := range strings.Split(string(buf[:n]), "\n") {
			if len(line) == 0 || line[0] != '{' {
				continue
			}
			callback(line)
		}
	}

	return nil
}

func scanFeature(client *redis.Client, ftype string) ([]*Feature, error) {
	cmd := redis.NewSliceCmd("SCAN", ftype)
	client.Process(cmd)
	res, err := cmd.Result()
	if err != nil {
		return nil, err
	}

	payload, _ := redis.NewSliceResult(res[1].([]interface{}), err).Result()
	features := make([]*Feature, len(payload))
	for i, item := range payload {
		itemRes, _ := redis.NewSliceResult(item.([]interface{}), nil).Result()
		features[i] = &Feature{itemRes[0].(string), itemRes[1].(string)}
	}
	return features, nil
}
