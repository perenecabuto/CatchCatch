// Code generated by mockery v1.0.0
package mocks

import mock "github.com/stretchr/testify/mock"
import websocket "github.com/perenecabuto/CatchCatch/server/websocket"

// WSEventHandler is an autogenerated mock type for the WSEventHandler type
type WSEventHandler struct {
	mock.Mock
}

// OnConnection provides a mock function with given fields: c
func (_m *WSEventHandler) OnConnection(c *websocket.WSConnListener) {
	_m.Called(c)
}
