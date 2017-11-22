package main

import (
	geo "github.com/kellydunn/golang-geo"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/model"
	gjson "github.com/tidwall/gjson"
	redis "gopkg.in/redis.v5"
)

// PlayerLocationService manage players and features
type PlayerLocationService interface {
	Register(p *model.Player) error
	Update(p *model.Player) error
	Remove(p *model.Player) error
	Players() (model.PlayerList, error)

	AddFeature(group, id, geojson string) (*model.Feature, error)
	FeatureByID(group, id string) (*model.Feature, error)
	Features(group string) ([]*model.Feature, error)
	FeaturesAround(group string, point *geo.Point) ([]*model.Feature, error)

	Clear()
}

// Tile38PlayerLocationService manages player locations
type Tile38PlayerLocationService struct {
	client *redis.Client
}

// NewPlayerLocationService build a PlayerLocationService
func NewPlayerLocationService(client *redis.Client) PlayerLocationService {
	return &Tile38PlayerLocationService{client}
}

// Register add new player
func (s *Tile38PlayerLocationService) Register(p *model.Player) error {
	return s.Update(p)
}

// Update player data
func (s *Tile38PlayerLocationService) Update(p *model.Player) error {
	cmd := redis.NewStringCmd("SET", "player", p.ID, "POINT", p.Lat, p.Lon)
	s.client.Process(cmd)
	return cmd.Err()
}

// Remove player
func (s *Tile38PlayerLocationService) Remove(p *model.Player) error {
	cmd := redis.NewStringCmd("DEL", "player", p.ID)
	s.client.Process(cmd)
	return cmd.Err()
}

// Players return all registered players
func (s *Tile38PlayerLocationService) Players() (model.PlayerList, error) {
	features, err := s.Features("player")
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

// AddFeature persist features
func (s *Tile38PlayerLocationService) AddFeature(group, id, geojson string) (*model.Feature, error) {
	cmd := redis.NewStringCmd("SET", group, id, "OBJECT", geojson)
	s.client.Process(cmd)
	if err := cmd.Err(); err != nil {
		return nil, err
	}
	return &model.Feature{ID: id, Coordinates: geojson, Group: group}, nil
}

// Features ...
func (s *Tile38PlayerLocationService) Features(group string) ([]*model.Feature, error) {
	cmd := redis.NewSliceCmd("SCAN", group)
	return featuresFromSliceCmd(s.client, group, cmd)
}

// Exists ...
func (s *Tile38PlayerLocationService) Exists(group, id string) (bool, error) {
	f, err := s.FeatureByID(group, id)
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return f != nil, nil
}

// FeatureByID ...
func (s *Tile38PlayerLocationService) FeatureByID(group, id string) (*model.Feature, error) {
	cmd := redis.NewStringCmd("GET", group, id)
	s.client.Process(cmd)
	coords, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	return &model.Feature{ID: id, Group: group, Coordinates: coords}, nil
}

// FeaturesAround return feature group near by point
func (s *Tile38PlayerLocationService) FeaturesAround(group string, point *geo.Point) ([]*model.Feature, error) {
	dist := 1000
	cmd := redis.NewSliceCmd("NEARBY", group, "POINT", point.Lat(), point.Lng(), dist)
	return featuresFromSliceCmd(s.client, group, cmd)
}

// Clear the database
func (s *Tile38PlayerLocationService) Clear() {
	s.client.FlushDb()
}

func featuresFromSliceCmd(client *redis.Client, group string, cmd *redis.SliceCmd) ([]*model.Feature, error) {
	client.Process(cmd)
	res, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	payload, _ := redis.NewSliceResult(res[1].([]interface{}), err).Result()
	features := make([]*model.Feature, len(payload))
	for i, item := range payload {
		itemRes, _ := redis.NewSliceResult(item.([]interface{}), nil).Result()
		features[i] = &model.Feature{ID: itemRes[0].(string), Coordinates: itemRes[1].(string), Group: group}
	}
	return features, nil
}
