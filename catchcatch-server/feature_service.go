package main

import (
	"context"

	"github.com/perenecabuto/CatchCatch/catchcatch-server/model"
)

type GeoFeatureService interface {
	FeaturesAroundPlayer(group string, player model.Player) ([]*model.Feature, error)
	FeaturesByGroup(group string) ([]*model.Feature, error)
	SetFeature(group, id, geojson string) error
	Clear() error

	ObservePlayersAround(context.Context, PlayersAroundCallback) error
	ObservePlayerNearToFeature(context.Context, string, PlayerNearToFeatureCallback) error
}

type PlayerNearToFeatureCallback func(playerID string, distTo float64, f model.Feature) error
type PlayersAroundCallback func(playerID string, movingPlayer model.Player, exit bool) error

func NewGeoFeatureService(repo Repository, stream EventStream) GeoFeatureService {
	return &Tile38GeoFeatureService{repo, stream}
}

type Tile38GeoFeatureService struct {
	repo   Repository
	stream EventStream
}

func (s *Tile38GeoFeatureService) FeaturesByGroup(group string) ([]*model.Feature, error) {
	return s.repo.Features(group)
}

func (s *Tile38GeoFeatureService) FeaturesAroundPlayer(group string, p model.Player) ([]*model.Feature, error) {
	return s.repo.FeaturesAround(group, p.Point())
}

func (s *Tile38GeoFeatureService) SetFeature(group, id, geojson string) error {
	_, err := s.repo.SetFeature(group, id, geojson)
	return err
}
func (s *Tile38GeoFeatureService) Clear() error {
	return s.repo.Clear()
}

func (s *Tile38GeoFeatureService) ObservePlayersAround(ctx context.Context, callback PlayersAroundCallback) error {
	return s.stream.StreamNearByEvents(ctx, "player", "player", "*", DefaultWatcherRange, func(d *Detection) error {
		playerID := d.NearByFeatID
		movingPlayer := model.Player{ID: d.FeatID, Lon: d.Lon, Lat: d.Lat}
		return callback(playerID, movingPlayer, d.Intersects == Exit)
	})
}

func (s *Tile38GeoFeatureService) ObservePlayerNearToFeature(ctx context.Context, group string, callback PlayerNearToFeatureCallback) error {
	return s.stream.StreamNearByEvents(ctx, group, "player", "*", DefaultWatcherRange, func(d *Detection) error {
		if d.Intersects == Inside {
			playerID := d.NearByFeatID
			f := model.Feature{ID: d.FeatID, Group: group, Coordinates: d.Coordinates}
			return callback(playerID, d.NearByMeters, f)
		}
		return nil
	})
}
