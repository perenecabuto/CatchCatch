package service

import (
	"context"
	"fmt"
	"sync"

	gjson "github.com/tidwall/gjson"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service/repository"
)

var ErrFeatureNotFound = repository.ErrFeatureNotFound

// TODO: renomear este service para user location service ou map service

// PlayerLocationService manage players and features
type PlayerLocationService interface {
	Set(p *model.Player) error
	Remove(playerID string) error
	All() (model.PlayerList, error)

	SetAdmin(id string, lat, lon float64) error
	RemoveAdmin(id string) error

	GeofenceByID(id string) (*model.Feature, error)
	Features() ([]*model.Feature, error)

	SetGeofence(id, coordinates string) error
	SetCheckpoint(id, coordinates string) error

	ObserveFeaturesEventsNearToAdmin(ctx context.Context, cb AdminNearToFeatureCallback) error
	ObservePlayersNearToGeofence(ctx context.Context, cb func(string, model.Player) error) error
	ObservePlayerNearToCheckpoint(context.Context, PlayerNearToFeatureCallback) error

	Clear() error
}

type PlayerNearToFeatureCallback func(playerID string, distTo float64, f model.Feature) error
type AdminNearToFeatureCallback func(adminID string, f model.Feature, action string) error

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

func (s *Tile38PlayerLocationService) SetAdmin(id string, lat, lon float64) error {
	_, err := s.repo.SetFeature("admin", id,
		fmt.Sprintf(`{"type": "Point", "coordinates": [%f, %f]}`, lon, lat))
	return err
}

func (s *Tile38PlayerLocationService) RemoveAdmin(id string) error {
	return s.repo.RemoveFeature("admin", id)
}

func (s *Tile38PlayerLocationService) GeofenceByID(id string) (*model.Feature, error) {
	return s.repo.FeatureByID("geofences", id)
}

func (s *Tile38PlayerLocationService) SetGeofence(id, coordinates string) error {
	_, err := s.repo.SetFeature("geofences", id, coordinates)
	return err
}

func (s *Tile38PlayerLocationService) SetCheckpoint(id, coordinates string) error {
	_, err := s.repo.SetFeature("checkpoint", id, coordinates)
	return err
}

func (s *Tile38PlayerLocationService) ObserveFeaturesEventsNearToAdmin(ctx context.Context, callback AdminNearToFeatureCallback) error {
	group := "admin"
	featureType := []string{"geofences", "checkpoint", "player"}
	errchan := make(chan error)

	myctx, cancel := context.WithCancel(ctx)

	var wg sync.WaitGroup
	wg.Add(len(featureType))
	for _, _ft := range featureType {
		go func(_ft string) {
			wg.Done()
			errchan <- s.stream.StreamNearByEvents(myctx, _ft, group, "*", DefaultGeoEventRange, func(d *repository.Detection) error {
				observerFeadID := d.NearByFeatID
				observedFeat := model.Feature{ID: d.FeatID, Group: _ft, Coordinates: d.Coordinates}
				return callback(observerFeadID, observedFeat, string(d.Intersects))
			})
		}(_ft)
	}
	wg.Wait()

	for {
		select {
		case err := <-errchan:
			cancel()
			return err
		case <-myctx.Done():
			cancel()
			return nil
		}
	}
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

// TODO: send player action (move, exit)
func (s *Tile38PlayerLocationService) ObservePlayersNearToGeofence(ctx context.Context, callback func(string, model.Player) error) error {
	return s.stream.StreamNearByEvents(ctx, "player", "geofences", "*", 100, func(d *repository.Detection) error {
		gameID := d.NearByFeatID
		if gameID == "" {
			return nil
		}
		p := model.Player{ID: d.FeatID, Lat: d.Lat, Lon: d.Lon}
		return callback(gameID, p)
	})
}

func (s *Tile38PlayerLocationService) Features() ([]*model.Feature, error) {
	featureType := []string{"geofences", "checkpoint", "player"}
	features := []*model.Feature{}
	for _, ft := range featureType {
		featsByType, err := s.repo.Features(ft)
		if err != nil {
			return nil, err
		}
		features = append(features, featsByType...)
	}
	return features, nil
}

func (s *Tile38PlayerLocationService) SetFeature(group, id, geojson string) error {
	_, err := s.repo.SetFeature(group, id, geojson)
	return err
}

func (s *Tile38PlayerLocationService) Clear() error {
	return s.repo.Clear()
}
