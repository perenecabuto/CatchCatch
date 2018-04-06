// Code generated by mockery v1.0.0
package mocks

import context "context"

import mock "github.com/stretchr/testify/mock"
import model "github.com/perenecabuto/CatchCatch/server/model"

// FeatureEventNearToAdmin is an autogenerated mock type for the FeatureEventNearToAdmin type
type FeatureEventNearToAdmin struct {
	mock.Mock
}

// OnFeatureEventNearToAdmin provides a mock function with given fields: _a0, _a1
func (_m *FeatureEventNearToAdmin) OnFeatureEventNearToAdmin(_a0 context.Context, _a1 func(string, model.Feature, string) error) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, func(string, model.Feature, string) error) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
