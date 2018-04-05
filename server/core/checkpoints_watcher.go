package core

import (
	"context"

	"github.com/perenecabuto/CatchCatch/server/service/messages"
	"github.com/tidwall/gjson"

	"github.com/golang/protobuf/proto"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/worker"
)

// CheckpointWatcher is responsible to watch players near to checkpoints
type CheckpointWatcher struct {
	messages messages.Dispatcher
	service  service.PlayerLocationService
}

// NewCheckpointWatcher creates a worker to watch checkpoints events
func NewCheckpointWatcher(m messages.Dispatcher, s service.PlayerLocationService) *CheckpointWatcher {
	return &CheckpointWatcher{m, s}
}

// ID implementation of worker.Worker.ID()
func (w *CheckpointWatcher) ID() string {
	return "CheckpointWatcher"
}

// Run listen to players near to checkpoints
func (w *CheckpointWatcher) Run(ctx context.Context, _ worker.TaskParams) error {
	return w.service.ObservePlayerNearToCheckpoint(ctx, func(playerID string, distTo float64, f model.Feature) error {
		lonlat := gjson.Get(f.Coordinates, "coordinates").Array()
		payload := &protobuf.Detection{
			EventName:    proto.String("checkpoint:detected"),
			Id:           &playerID,
			FeatId:       &f.ID,
			NearByFeatId: &playerID,
			NearByMeters: &distTo,
			Lon:          proto.Float64(lonlat[0].Float()),
			Lat:          proto.Float64(lonlat[1].Float()),
		}
		data, _ := proto.Marshal(payload)
		err := w.messages.Publish("checkpoint:detected", data)
		if err != nil {
			return err
		}
		return nil
	})
}

// OnCheckpointNearToPlayer notify about players near to checkpoints
func (w *CheckpointWatcher) OnCheckpointNearToPlayer(ctx context.Context, cb func(*protobuf.Detection) error) error {
	return w.messages.Subscribe(ctx, "checkpoint:detected", func(data []byte) error {
		payload := &protobuf.Detection{}
		err := proto.Unmarshal(data, payload)
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		cb(payload)
		return nil
	})
}
