// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	context "context"

	candishared "github.com/golangid/candi/candishared"

	mock "github.com/stretchr/testify/mock"
)

// Publisher is an autogenerated mock type for the Publisher type
type Publisher struct {
	mock.Mock
}

// PublishMessage provides a mock function with given fields: ctx, args
func (_m *Publisher) PublishMessage(ctx context.Context, args *candishared.PublisherArgument) error {
	ret := _m.Called(ctx, args)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *candishared.PublisherArgument) error); ok {
		r0 = rf(ctx, args)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
