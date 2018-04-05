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

// FeaturesEventsWatcher is responsible to watch to Features' CRUD events
type FeaturesEventsWatcher struct {
	messages messages.Dispatcher
	players  service.PlayerLocationService
}

// NewFeaturesEventsWatcher creates a new feature watcher \o/
func NewFeaturesEventsWatcher(m messages.Dispatcher, p service.PlayerLocationService) *FeaturesEventsWatcher {
	return &FeaturesEventsWatcher{m, p}
}

// ID implementation of worker.Worker.ID()
func (w *FeaturesEventsWatcher) ID() string {
	return "FeaturesEventsWatcher"
}

// FeatureEventsNearToAdminPayload wraps
type FeatureEventsNearToAdminPayload struct {
	AdminID string
	Feature model.Feature
	Action  string
}

// Run watcher feature events near to admins
func (w *FeaturesEventsWatcher) Run(ctx context.Context, _ worker.TaskParams) error {
	return w.players.ObserveFeaturesEventsNearToAdmin(ctx, func(id string, f model.Feature, action string) error {
		data, _ := json.Marshal(FeatureEventsNearToAdminPayload{AdminID: id, Feature: f, Action: action})
		err := w.messages.Publish(featuresMessageTopic, data)
		if err != nil {
			log.Println("AdminHandler:WatchGeofences:", err.Error())
		}
		return nil
	})
}

// OnFeatureEventNearToAdmin notify about feature event near to admins
func (w *FeaturesEventsWatcher) OnFeatureEventNearToAdmin(ctx context.Context, cb func(adminID string, feat model.Feature, action string) error) error {
	return w.messages.Subscribe(ctx, featuresMessageTopic, func(data []byte) error {
		payload := &FeatureEventsNearToAdminPayload{}
		err := json.Unmarshal(data, payload)
		if err != nil {
			log.Println("[FeaturesEventsWatcher] ObserveFeaturesEventsNearToAdmin unmarshal error:", err)
			return nil
		}
		err = cb(payload.AdminID, payload.Feature, payload.Action)
		if err != nil {
			log.Println("[AdminHandler] WatchFeatureEvents:exiting:callback error:", err)
		}
		return nil
	})
}
