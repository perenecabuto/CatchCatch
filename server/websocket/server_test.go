package websocket_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/perenecabuto/CatchCatch/server/websocket"
	"github.com/perenecabuto/CatchCatch/server/websocket/mocks"
)

func TestWSServer_Listen(t *testing.T) {
	tests := []struct {
		name           string
		driver         *mocks.WSDriver
		eventHandler   *mocks.WSEventHandler
		assertError    assert.ErrorAssertionFunc
		assertResponse assert.ValueAssertionFunc
	}{
		{
			"it return error when handler fail to start",
			&mocks.WSDriver{},
			func() *mocks.WSEventHandler {
				h := &mocks.WSEventHandler{}
				h.On("OnStart", mock.Anything, mock.Anything).Return(errors.New(""))
				return h
			}(),
			assert.Error,
			assert.Nil,
		},
		{
			"it return a HTTPHandler after handler starts",
			func() *mocks.WSDriver {
				d := &mocks.WSDriver{}
				h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				})
				d.On("HTTPHandler", mock.Anything, mock.Anything).Return(h)
				return d
			}(),
			func() *mocks.WSEventHandler {
				h := &mocks.WSEventHandler{}
				h.On("OnStart", mock.Anything, mock.Anything).Return(nil)
				return h
			}(),
			assert.NoError,
			assert.NotNil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			wss := websocket.NewWSServer(tt.driver, tt.eventHandler)
			got, err := wss.Listen(ctx)
			tt.assertError(t, err)
			tt.assertResponse(t, got)
			tt.eventHandler.AssertExpectations(t)
		})
	}
}
