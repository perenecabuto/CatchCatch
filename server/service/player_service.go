package service

import (
	"context"
	"fmt"

	gjson "github.com/tidwall/gjson"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service/repository"
)

// TODO: mudar este service para user location service ou map service
// TODO: crud de admin (ou user com roles)

// PlayerLocationService manage players and features
type PlayerLocationService interface {
	Set(p *model.Player) error
	Remove(playerID string) error
	All() (model.PlayerList, error)

	GeofenceByID(id string) (*model.Feature, error)

	ObservePlayersAround(context.Context, PlayersAroundCallback) error
	ObservePlayerNearToFeature(context.Context, string, PlayerNearToFeatureCallback) error

	ObservePlayersInsideGeofence(ctx context.Context, callback func(string, model.Player) error) error
	ObservePlayerNearToCheckpoint(context.Context, PlayerNearToFeatureCallback) error
}

type PlayerNearToFeatureCallback func(playerID string, distTo float64, f model.Feature) error
type PlayersAroundCallback func(playerID string, movingPlayer model.Player, exit bool) error

// Tile38PlayerLocationService manages player locations
type Tile38PlayerLocationService struct {
	repo   repository.Repository
	stream repository.EventStream
}

// NewPlayerLocationService build a PlayerLocationService
func NewPlayerLocationService(repo repository.Repository, stream repository.EventStream) PlayerLocationService {
	return &Tile38PlayerLocationService{repo, stream}
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
func (s *Tile38PlayerLocationService) Remove(playerID string) error {
	return s.repo.RemoveFeature("player", playerID)
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

func (s *Tile38PlayerLocationService) GeofenceByID(id string) (*model.Feature, error) {
	return s.repo.FeatureByID("geofences", id)
}

func (s *Tile38PlayerLocationService) ObservePlayersAround(ctx context.Context, callback PlayersAroundCallback) error {
	return s.stream.StreamNearByEvents(ctx, "player", "player", "*", DefaultGeoEventRange, func(d *repository.Detection) error {
		playerID := d.NearByFeatID
		movingPlayer := model.Player{ID: d.FeatID, Lon: d.Lon, Lat: d.Lat}
		return callback(playerID, movingPlayer, d.Intersects == repository.Exit)
	})
}

func (s *Tile38PlayerLocationService) ObservePlayerNearToFeature(ctx context.Context, group string, callback PlayerNearToFeatureCallback) error {
	return s.stream.StreamNearByEvents(ctx, group, "player", "*", DefaultGeoEventRange, func(d *repository.Detection) error {
		if d.Intersects == repository.Inside {
			playerID := d.NearByFeatID
			f := model.Feature{ID: d.FeatID, Group: group, Coordinates: d.Coordinates}
			return callback(playerID, d.NearByMeters, f)
		}
		return nil
	})
}

func (s *Tile38PlayerLocationService) ObservePlayerNearToCheckpoint(ctx context.Context, callback PlayerNearToFeatureCallback) error {
	return s.stream.StreamNearByEvents(ctx, "player", "checkpoint", "*", DefaultGeoEventRange, func(d *repository.Detection) error {
		if d.Intersects == repository.Inside {
			playerID := d.FeatID
			f := model.Feature{ID: d.FeatID, Group: "checkpoint", Coordinates: d.Coordinates}
			return callback(playerID, d.NearByMeters, f)
		}
		return nil
	})
}

func (s *Tile38PlayerLocationService) ObservePlayersInsideGeofence(ctx context.Context, callback func(string, model.Player) error) error {
	return s.stream.StreamNearByEvents(ctx, "player", "geofences", "*", 1, func(d *repository.Detection) error {
		gameID := d.NearByFeatID
		if gameID == "" {
			return nil
		}
		p := model.Player{ID: d.FeatID, Lat: d.Lat, Lon: d.Lon}
		return callback(gameID, p)
	})
}
