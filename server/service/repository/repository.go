package repository

import (
	"github.com/perenecabuto/CatchCatch/server/model"

	redis "github.com/go-redis/redis"
	geo "github.com/kellydunn/golang-geo"
)

const (
	// ErrFeatureNotFound happens when feature does not exists in the storage
	ErrFeatureNotFound = redis.Nil
)

// Repository is the geospatial repository
type Repository interface {
	SetFeature(group, id, geojson string) (*model.Feature, error)
	Expire(group, id string, expireInSecs int) error
	Exists(group, id string) (bool, error)
	FeatureByID(group, id string) (*model.Feature, error)
	RemoveFeature(group, id string) error
	Features(group string) ([]*model.Feature, error)
	FeaturesAround(group string, point *geo.Point) ([]*model.Feature, error)
	FeatureExtraData(group, id string) (string, error)
	SetFeatureExtraData(group, id string, j string) error
	DelFeatureExtraData(group, id string) error
	Clear() error
}

// Tile38Repository tile38 implementation of Repository
type Tile38Repository struct {
	client *redis.Client
}

// NewRepository creates a new tile38 repository
func NewRepository(client *redis.Client) Repository {
	return &Tile38Repository{client}
}

// SetFeature persist feature
func (r *Tile38Repository) SetFeature(group, id, geojson string) (*model.Feature, error) {
	cmd := redis.NewStringCmd("SET", group, id, "OBJECT", geojson)
	r.client.Process(cmd)
	if err := cmd.Err(); err != nil {
		return nil, err
	}
	return &model.Feature{ID: id, Coordinates: geojson, Group: group}, nil
}

// Expire set feature expiration time in secs
func (r *Tile38Repository) Expire(group, id string, expireInSecs int) error {
	cmd := redis.NewStringCmd("EXPIRE", group, id, expireInSecs)
	return r.client.Process(cmd)
}

// RemoveFeature removes a feature
func (r *Tile38Repository) RemoveFeature(group, id string) error {
	cmd := redis.NewStringCmd("DEL", group, id)
	r.client.Process(cmd)
	return cmd.Err()
}

// Features ...
func (r *Tile38Repository) Features(group string) ([]*model.Feature, error) {
	cmd := redis.NewSliceCmd("SCAN", group)
	return featuresFromSliceCmd(r.client, group, cmd)
}

// Exists ...
func (r *Tile38Repository) Exists(group, id string) (bool, error) {
	f, err := r.FeatureByID(group, id)
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return f != nil, nil
}

// FeatureByID ...
func (r *Tile38Repository) FeatureByID(group, id string) (*model.Feature, error) {
	cmd := redis.NewStringCmd("GET", group, id)
	r.client.Process(cmd)
	coords, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	return &model.Feature{ID: id, Group: group, Coordinates: coords}, nil
}

// FeatureExtraData ...
func (r *Tile38Repository) FeatureExtraData(group, id string) (string, error) {
	cmd := redis.NewStringCmd("GET", group, id+":extra")
	r.client.Process(cmd)
	return cmd.Val(), cmd.Err()
}

// SetFeatureExtraData ...
func (r *Tile38Repository) SetFeatureExtraData(group, id, data string) error {
	cmd := redis.NewStringCmd("SET", group, id+":extra", "STRING", data)
	r.client.Process(cmd)
	return cmd.Err()
}

// DelFeatureExtraData ...
func (r *Tile38Repository) DelFeatureExtraData(group, id string) error {
	cmd := redis.NewStringCmd("DEL", group, id+":extra")
	r.client.Process(cmd)
	return cmd.Err()
}

// FeaturesAround return feature group near by point
func (r *Tile38Repository) FeaturesAround(group string, point *geo.Point) ([]*model.Feature, error) {
	dist := 1000
	cmd := redis.NewSliceCmd("NEARBY", group, "POINT", point.Lat(), point.Lng(), dist)
	return featuresFromSliceCmd(r.client, group, cmd)
}

// Clear the database
func (r *Tile38Repository) Clear() error {
	return r.client.FlushDb().Err()
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
		features[i] = &model.Feature{ID: itemRes[0].(string), Group: group, Coordinates: itemRes[1].(string)}
	}
	return features, nil
}
