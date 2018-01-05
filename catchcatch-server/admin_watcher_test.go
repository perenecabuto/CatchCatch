package main

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	geo "github.com/kellydunn/golang-geo"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/model"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/protobuf"
)

func TestNewAdminWatcher(t *testing.T) {
	if w := createAdminWatcher(); w == nil {
		t.Fatal("Can't create AdminWatcher")
	}
}

func TestWatchCheckPointsMustNotifyPlayersNearToCheckPoinstsTheDistToIt(t *testing.T) {
	w := createAdminWatcher()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := w.WatchCheckpoints(ctx)
	if err != nil {
		t.Fatal(err)
	}

	playerID := "player1"
	distToCheckPoint := 12.0
	checkPoint := model.Feature{Group: "checkpoint", ID: "checkpoint1"}

	wss := w.wss
	c := &MockWSConnection{}
	cListener := wss.Add(c)
	wss.connections[playerID] = cListener

	geoS := w.service.(*MockGeoService)
	geoS.PlayerNearToFeatureCallback(playerID, distToCheckPoint, checkPoint)

	if len(c.messages) == 0 {
		t.Fatal("Expected checkpoint event to be sent")
	}
	expected, _ := proto.Marshal(&protobuf.Detection{
		EventName:    proto.String("checkpoint:detected"),
		Id:           &checkPoint.ID,
		FeatId:       &checkPoint.ID,
		NearByFeatId: &playerID,
		NearByMeters: &distToCheckPoint,
	})
	if !reflect.DeepEqual(c.messages[0], expected) {
		t.Fatal("Diff:", expected, c.messages[0])
	}
}

type MockWSDriver struct{}

func (d *MockWSDriver) Handler(ctx context.Context, onConnect func(context.Context, WSConnection)) http.Handler {
	return nil
}

type MockWSConnection struct {
	messages [][]byte
}

func (c *MockWSConnection) Read(*[]byte) (int, error) {
	return 0, nil
}
func (c *MockWSConnection) Send(payload []byte) error {
	c.messages = append(c.messages, payload)
	return nil
}
func (c *MockWSConnection) Close() error {
	return nil
}

func createAdminWatcher() *AdminWatcher {
	wss := NewWSServer(&MockWSDriver{})
	// repo := &MockRepository{}
	// stream := &MockEventStream{}
	geoService := &MockGeoService{}
	return NewAdminWatcher(geoService, wss)
}

type MockGeoService struct {
	PlayersAroundCallback       PlayersAroundCallback
	PlayerNearToFeatureCallback PlayerNearToFeatureCallback
}

func (gs *MockGeoService) FeaturesAroundPlayer(group string, player model.Player) ([]*model.Feature, error) {
	return nil, nil
}
func (gs *MockGeoService) FeaturesByGroup(group string) ([]*model.Feature, error) {
	return nil, nil
}
func (gs *MockGeoService) SetFeature(group, id, geojson string) error {
	return nil
}
func (gs *MockGeoService) Clear() error {
	return nil
}
func (gs *MockGeoService) ObservePlayersAround(_ context.Context, cb PlayersAroundCallback) error {
	gs.PlayersAroundCallback = cb
	return nil
}
func (gs *MockGeoService) ObservePlayerNearToFeature(_ context.Context, _ string, cb PlayerNearToFeatureCallback) error {
	gs.PlayerNearToFeatureCallback = cb
	return nil
}

type MockEventStream struct{}

func (es *MockEventStream) StreamNearByEvents(
	ctx context.Context, nearByKey, roamKey, roamID string, meters int, callback DetectionHandler) error {
	return nil
}
func (es *MockEventStream) StreamIntersects(
	ctx context.Context, intersectKey, onKey, onKeyID string, callback DetectionHandler) error {
	return nil
}

type MockRepository struct{}

func (r *MockRepository) SetFeature(group, id, geojson string) (*model.Feature, error) {
	return nil, nil
}
func (r *MockRepository) Exists(group, id string) (bool, error) {
	return false, nil
}
func (r *MockRepository) FeatureByID(group, id string) (*model.Feature, error) {
	return nil, nil
}
func (r *MockRepository) RemoveFeature(group, id string) error {
	return nil
}
func (r *MockRepository) Features(group string) ([]*model.Feature, error) {
	return nil, nil
}
func (r *MockRepository) FeaturesAround(group string, point *geo.Point) ([]*model.Feature, error) {
	return nil, nil
}
func (r *MockRepository) FeatureExtraData(group, id string) (string, error) {
	return "", nil
}
func (r *MockRepository) SetFeatureExtraData(group, id string, j string) error {
	return nil
}
func (r *MockRepository) DelFeatureExtraData(group, id string) error {
	return nil
}
func (r *MockRepository) Clear() error {
	return nil
}
