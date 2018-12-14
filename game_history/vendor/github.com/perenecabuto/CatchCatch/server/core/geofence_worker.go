package core

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/service/messages"
	"github.com/perenecabuto/CatchCatch/server/worker"
	"github.com/pkg/errors"
)

const (
	geofencesEventsTopic = "geofences:player:near:events"
)

// GeofenceEventsWorker listen to player over geofences and start GameWorkers
type GeofenceEventsWorker struct {
	service  service.PlayerLocationService
	workers  worker.Manager
	messages messages.Dispatcher
}

var _ worker.Task = &GeofenceEventsWorker{}

// NewGeofenceEventsWorker creates Geo
func NewGeofenceEventsWorker(s service.PlayerLocationService, m worker.Manager, d messages.Dispatcher) *GeofenceEventsWorker {
	return &GeofenceEventsWorker{s, m, d}
}

// ID implements worker.Worker ID
func (gw *GeofenceEventsWorker) ID() string {
	return "GeofenceEventsWorker"
}

// OnPlayerNearToGeofence listen to player near to geofences
func (gw *GeofenceEventsWorker) OnPlayerNearToGeofence(ctx context.Context, fn func(p model.Player, g service.GameWithCoords) error) error {
	return gw.messages.Subscribe(ctx, geofencesEventsTopic, func(msg []byte) error {
		gameID := "test"
		p := model.Player{}
		g := service.GameWithCoords{Game: game.NewGame(gameID)}
		err := fn(p, g)
		return errors.Cause(err)
	})
}

// Run listen to player over geofences and start GameWorkers
func (gw *GeofenceEventsWorker) Run(ctx context.Context, _ worker.TaskParams) error {
	return gw.service.ObservePlayersNearToGeofence(ctx, func(id string, p model.Player) error {
		// TODO: set player already notified with an time stamp to check when notify it again
		// TODO: do not notify player already notified and return
		f, err := gw.service.GeofenceByID(id)
		if err != nil {
			return err
		}
		data, err := json.Marshal(f)
		if err != nil {
			return errors.Wrapf(err, "can't encode feature")
		}
		err = gw.messages.Publish(geofencesEventsTopic, data)
		if err != nil {
			return errors.Wrapf(err, "can't publish feature")
		}
		err = gw.workers.RunUnique(GameWorker{},
			worker.TaskParams{"gameID": id, "coordinates": f.Coordinates},
			fmt.Sprintf("game:%s", id))
		return errors.Wrapf(err, "can't start game worker")
	})
}
