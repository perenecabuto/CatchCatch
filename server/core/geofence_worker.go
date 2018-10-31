package core

import (
	"context"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/worker"
)

// GeofenceEventsWorker listen to player over geofences and start GameWorkers
type GeofenceEventsWorker struct {
	service service.PlayerLocationService
	workers worker.Manager
}

// NewGeofenceEventsWorker creates Geo
func NewGeofenceEventsWorker(s service.PlayerLocationService, m worker.Manager) worker.Worker {
	return &GeofenceEventsWorker{s, m}
}

// ID implements worker.Worker ID
func (gw GeofenceEventsWorker) ID() string {
	return "GeofenceEventsWorker"
}

// Run listen to player over geofences and start GameWorkers
func (gw GeofenceEventsWorker) Run(ctx context.Context, _ worker.TaskParams) error {
	return gw.service.ObservePlayersNearToGeofence(ctx, func(id string, _ model.Player) error {
		f, err := gw.service.GeofenceByID(id)
		if err != nil {
			return err
		}
		return gw.workers.RunUnique(GameWorker{}, worker.TaskParams{"gameID": id, "coordinates": f.Coordinates})
	})
}
