package main

import (
	"errors"
	"fmt"

	geo "github.com/kellydunn/golang-geo"
	gjson "github.com/tidwall/gjson"
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
	ID  string  `json:"id"`
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

func (p *Player) String() string {
	return fmt.Sprintln("id:", p.ID, "lat:", p.Lat, "lon:", p.Lon)
}

// Point returns geo.Point with coordinates
func (p *Player) Point() *geo.Point {
	return geo.NewPoint(p.Lat, p.Lon)
}

// DistTo returns the distance to other player
func (p *Player) DistTo(other *Player) float64 {
	return p.Point().GreatCircleDistance(other.Point()) * 1000
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
	cmd := redis.NewStringCmd("SET", "player", p.ID, "POINT", p.Lat, p.Lon)
	s.client.Process(cmd)
	return cmd.Err()
}

// Remove player
func (s *PlayerLocationService) Remove(p *Player) error {
	cmd := redis.NewStringCmd("DEL", "player", p.ID)
	s.client.Process(cmd)
	return cmd.Err()
}

// Players return all registered players
func (s *PlayerLocationService) Players() (*PlayerList, error) {
	features, err := s.Features("player")
	if err != nil {
		return nil, err
	}
	list := &PlayerList{make([]*Player, len(features))}
	for i, f := range features {
		coords := gjson.Get(f.Coordinates, "coordinates").Array()
		list.Players[i] = &Player{ID: f.ID, Lat: coords[1].Float(), Lon: coords[0].Float()}
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
	coords := gjson.Get(data, "coordinates").Array()
	return &Player{ID: id, Lat: coords[1].Float(), Lon: coords[0].Float()}, nil
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
	cmd := redis.NewSliceCmd("SCAN", group)
	return featuresFromSliceCmd(s.client, group, cmd)
}

// FeaturesAround return feature group near by point
func (s *PlayerLocationService) FeaturesAround(group string, point *geo.Point) ([]*Feature, error) {
	dist := 1000
	cmd := redis.NewSliceCmd("NEARBY", group, "POINT", point.Lat(), point.Lng(), dist)
	return featuresFromSliceCmd(s.client, group, cmd)
}

func featuresFromSliceCmd(client *redis.Client, group string, cmd *redis.SliceCmd) ([]*Feature, error) {
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
