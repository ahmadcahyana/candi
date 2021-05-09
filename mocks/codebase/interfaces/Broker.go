// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	context "context"

	interfaces "github.com/golangid/candi/codebase/interfaces"
	mock "github.com/stretchr/testify/mock"

	types "github.com/golangid/candi/codebase/factory/types"
)

// Broker is an autogenerated mock type for the Broker type
type Broker struct {
	mock.Mock
}

// Disconnect provides a mock function with given fields: ctx
func (_m *Broker) Disconnect(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetConfiguration provides a mock function with given fields: _a0
func (_m *Broker) GetConfiguration(_a0 types.Worker) interface{} {
	ret := _m.Called(_a0)

	var r0 interface{}
	if rf, ok := ret.Get(0).(func(types.Worker) interface{}); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	return r0
}

// Health provides a mock function with given fields:
func (_m *Broker) Health() map[string]error {
	ret := _m.Called()

	var r0 map[string]error
	if rf, ok := ret.Get(0).(func() map[string]error); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]error)
		}
	}

	return r0
}

// Publisher provides a mock function with given fields: _a0
func (_m *Broker) Publisher(_a0 types.Worker) interfaces.Publisher {
	ret := _m.Called(_a0)

	var r0 interfaces.Publisher
	if rf, ok := ret.Get(0).(func(types.Worker) interfaces.Publisher); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interfaces.Publisher)
		}
	}

	return r0
}
