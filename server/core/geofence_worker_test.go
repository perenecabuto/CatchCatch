package core_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/perenecabuto/CatchCatch/server/core"
	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
	wmocks "github.com/perenecabuto/CatchCatch/server/worker/mocks"
)

func TestGeofenceWorkerNofityWorkerManagerToRunGameWorker(t *testing.T) {
	service := &smocks.PlayerLocationService{}
	manager := &wmocks.Manager{}
	worker := core.NewGeofenceEventsWorker(service, manager)

	service.On("ObservePlayersNearToGeofence", mock.Anything, mock.Anything).Return(nil)

	worker.Run(context.Background(), nil)
}
