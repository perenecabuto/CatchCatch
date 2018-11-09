package core_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/perenecabuto/CatchCatch/server/core"
	smocks "github.com/perenecabuto/CatchCatch/server/service/mocks"
	wmocks "github.com/perenecabuto/CatchCatch/server/worker/mocks"
)

func TestGeofenceWorkerNofityWorkerManagerToRunGameWorker(t *testing.T) {
	service := &smocks.PlayerLocationService{}
	messages := &smocks.Dispatcher{}
	manager := &wmocks.Manager{}
	worker := core.NewGeofenceEventsWorker(service, manager, messages)

	service.On("ObservePlayersNearToGeofence", mock.Anything, mock.Anything).Return(nil)

	err := worker.Run(context.Background(), nil)
	require.NoError(t, err)
}
