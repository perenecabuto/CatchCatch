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
	service := new(smocks.PlayerLocationService)
	manager := new(wmocks.Manager)
	worker := core.NewGeofenceEventsWorker(service, manager)

	service.On("ObservePlayersInsideGeofence", mock.Anything, mock.Anything).Return(nil)

	worker.Run(context.Background(), nil)
}
