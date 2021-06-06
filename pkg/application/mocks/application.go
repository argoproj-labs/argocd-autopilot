// Code generated by mockery v1.1.1. DO NOT EDIT.

package mocks

import (
	fs "github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	mock "github.com/stretchr/testify/mock"
)

// Application is an autogenerated mock type for the Application type
type Application struct {
	mock.Mock
}

// CreateFiles provides a mock function with given fields: repofs, projectName
func (_m *Application) CreateFiles(repofs fs.FS, projectName string) error {
	ret := _m.Called(repofs, projectName)

	var r0 error
	if rf, ok := ret.Get(0).(func(fs.FS, string) error); ok {
		r0 = rf(repofs, projectName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Name provides a mock function with given fields:
func (_m *Application) Name() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
