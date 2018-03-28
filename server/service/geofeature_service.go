package service

import (
	geo "github.com/kellydunn/golang-geo"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service/repository"
)

const ErrFeatureNotFound = repository.ErrFeatureNotFound

// TODO: tornar menos generico
// TODO: ouvir, buscar e criar geofences

type GeoFeatureService interface {
	FeaturesAroundPoint(group string, point *geo.Point) ([]*model.Feature, error)
	FeaturesByGroup(group string) ([]*model.Feature, error)
	SetFeature(group, id, geojson string) error
	Clear() error
}

func NewGeoFeatureService(repo repository.Repository) GeoFeatureService {
	return &Tile38GeoFeatureService{repo}
}

type Tile38GeoFeatureService struct {
	repo repository.Repository
}

func (s *Tile38GeoFeatureService) FeaturesByGroup(group string) ([]*model.Feature, error) {
	return s.repo.Features(group)
}

func (s *Tile38GeoFeatureService) FeaturesAroundPoint(group string, point *geo.Point) ([]*model.Feature, error) {
	return s.repo.FeaturesAround(group, point)
}

func (s *Tile38GeoFeatureService) SetFeature(group, id, geojson string) error {
	_, err := s.repo.SetFeature(group, id, geojson)
	return err
}
func (s *Tile38GeoFeatureService) Clear() error {
	return s.repo.Clear()
}
