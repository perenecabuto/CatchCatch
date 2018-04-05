package core

import (
	"context"
	"encoding/json"
	"log"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/service/messages"
	"github.com/perenecabuto/CatchCatch/server/worker"
)

const featuresMessageTopic = "admin:feature:event"

type FeaturesEventsWatcher struct {
	messages messages.Dispatcher
	players  service.PlayerLocationService
}

func NewFeaturesEventsWatcher(m messages.Dispatcher, p service.PlayerLocationService) *FeaturesEventsWatcher {
	return &FeaturesEventsWatcher{m, p}
}

func (w *FeaturesEventsWatcher) ID() string {
	return "FeaturesEventsWatcher"
}

type EventsNearToAdminPayload struct {
	AdminID string
	Feature model.Feature
	Action  string
}

func (w *FeaturesEventsWatcher) Run(ctx context.Context, _ worker.TaskParams) error {
	return w.players.ObserveFeaturesEventsNearToAdmin(ctx, func(id string, f model.Feature, action string) error {
		data, _ := json.Marshal(EventsNearToAdminPayload{AdminID: id, Feature: f, Action: action})
		err := w.messages.Publish(featuresMessageTopic, data)
		if err != nil {
			log.Println("AdminHandler:WatchGeofences:", err.Error())
		}
		return nil
	})
}

func (w *FeaturesEventsWatcher) ObserveFeaturesEventsNearToAdmin(ctx context.Context, cb func(adminID string, feat model.Feature, action string) error) error {
	stream := make(chan []byte)
	err := w.messages.Subscribe(ctx, featuresMessageTopic, func(data []byte) error {
		stream <- data
		return nil
	})
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case data := <-stream:
			payload := &EventsNearToAdminPayload{}
			err := json.Unmarshal(data, payload)
			if err != nil {
				log.Println("[FeaturesEventsWatcher] ObserveFeaturesEventsNearToAdmin:", err)
				continue
			}
			err = cb(payload.AdminID, payload.Feature, payload.Action)
			if err != nil {
				log.Println("[AdminHandler] WatchFeatureEvents:exiting:callback error:", err)
				return err
			}
			return nil
		}
	}
}
