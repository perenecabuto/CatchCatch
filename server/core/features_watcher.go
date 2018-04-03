package core

import (
	"context"
	"log"

	"github.com/tidwall/sjson"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/service/messages"
	"github.com/perenecabuto/CatchCatch/server/worker"
)

const FeaturesMessageTopic = "admin:feature:event"

type FeaturesEventsWatcher struct {
	messages messages.Dispatcher
	players  service.PlayerLocationService
}

func NewFeaturesEventsWatcher(m messages.Dispatcher, p service.PlayerLocationService) worker.Worker {
	return &FeaturesEventsWatcher{m, p}
}

func (w *FeaturesEventsWatcher) ID() string {
	return "FeaturesEventsWatcher"
}

func (w *FeaturesEventsWatcher) Run(ctx context.Context, _ worker.TaskParams) error {
	return w.players.ObserveFeaturesEventsNearToAdmin(ctx, func(id string, f model.Feature, action string) error {
		data, _ := sjson.SetBytes([]byte{}, "id", id)
		data, _ = sjson.SetBytes(data, "featID", f.ID)
		data, _ = sjson.SetBytes(data, "group", f.Group)
		data, _ = sjson.SetBytes(data, "coordinates", f.Coordinates)
		data, _ = sjson.SetBytes(data, "action", action)
		err := w.messages.Publish(FeaturesMessageTopic, data)
		if err != nil {
			log.Println("AdminHandler:WatchGeofences:", err.Error())
		}
		return nil
	})
}
