package core

import (
	"context"
	"log"

	"github.com/perenecabuto/CatchCatch/server/service/messages"
	"github.com/perenecabuto/CatchCatch/server/websocket"
	"github.com/tidwall/gjson"

	"github.com/golang/protobuf/proto"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/worker"
)

type checkpointsWatcher struct {
	wss      *websocket.WSServer
	messages messages.Dispatcher
	service  service.PlayerLocationService
}

// NewCheckpointWatcher creates a worker to watch checkpoints events
func NewCheckpointWatcher(wss *websocket.WSServer, m messages.Dispatcher, s service.PlayerLocationService) worker.Worker {
	return &checkpointsWatcher{wss, m, s}
}

func (w *checkpointsWatcher) ID() string {
	return "checkpointsWatcher"
}

// Run watches checkpoints events
func (w *checkpointsWatcher) Run(ctx context.Context, _ worker.TaskParams) error {
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
		// data, _ := proto.Marshal(payload)
		// err := w.messages.Publish("checkpoint:detected", data)
		// if err != nil {
		// 	return err
		// }
		// TODO: remove this shit from HERE
		// it's a handler resposibility
		if err := w.wss.Emit(playerID, payload); err != nil {
			log.Println("AdminWatcher:WatchCheckpoints:", playerID, err.Error())
		}
		return nil
	})
}
