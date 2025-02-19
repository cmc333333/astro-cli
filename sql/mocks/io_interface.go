// Code generated by mockery v2.16.0. DO NOT EDIT.

package mocks

import (
	io "io"

	mock "github.com/stretchr/testify/mock"
)

// IoBind is an autogenerated mock type for the IoBind type
type IoBind struct {
	mock.Mock
}

// Copy provides a mock function with given fields: dst, src
func (_m *IoBind) Copy(dst io.Writer, src io.Reader) (int64, error) {
	ret := _m.Called(dst, src)

	var r0 int64
	if rf, ok := ret.Get(0).(func(io.Writer, io.Reader) int64); ok {
		r0 = rf(dst, src)
	} else {
		r0 = ret.Get(0).(int64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(io.Writer, io.Reader) error); ok {
		r1 = rf(dst, src)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewIoBind interface {
	mock.TestingT
	Cleanup(func())
}

// NewIoBind creates a new instance of IoBind. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIoBind(t mockConstructorTestingTNewIoBind) *IoBind {
	mock := &IoBind{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
