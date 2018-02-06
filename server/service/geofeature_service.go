package service

import (
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service/repository"
)

const ErrFeatureNotFound = repository.ErrFeatureNotFound

type GeoFeatureService interface {
	FeaturesAroundPlayer(group string, player model.Player) ([]*model.Feature, error)
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
