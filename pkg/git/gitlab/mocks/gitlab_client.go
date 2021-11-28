// Code generated by mockery (devel). DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	gitlab "github.com/xanzy/go-gitlab"
)

// GitlabClient is an autogenerated mock type for the GitlabClient type
type GitlabClient struct {
	mock.Mock
}

// CreateProject provides a mock function with given fields: opt, options
func (_m *GitlabClient) CreateProject(opt *gitlab.CreateProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, opt)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *gitlab.Project
	if rf, ok := ret.Get(0).(func(*gitlab.CreateProjectOptions, ...gitlab.RequestOptionFunc) *gitlab.Project); ok {
		r0 = rf(opt, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*gitlab.Project)
		}
	}

	var r1 *gitlab.Response
	if rf, ok := ret.Get(1).(func(*gitlab.CreateProjectOptions, ...gitlab.RequestOptionFunc) *gitlab.Response); ok {
		r1 = rf(opt, options...)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*gitlab.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*gitlab.CreateProjectOptions, ...gitlab.RequestOptionFunc) error); ok {
		r2 = rf(opt, options...)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// CurrentUser provides a mock function with given fields: options
func (_m *GitlabClient) CurrentUser(options ...gitlab.RequestOptionFunc) (*gitlab.User, *gitlab.Response, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *gitlab.User
	if rf, ok := ret.Get(0).(func(...gitlab.RequestOptionFunc) *gitlab.User); ok {
		r0 = rf(options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*gitlab.User)
		}
	}

	var r1 *gitlab.Response
	if rf, ok := ret.Get(1).(func(...gitlab.RequestOptionFunc) *gitlab.Response); ok {
		r1 = rf(options...)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*gitlab.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(...gitlab.RequestOptionFunc) error); ok {
		r2 = rf(options...)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ListGroups provides a mock function with given fields: opt, options
func (_m *GitlabClient) ListGroups(opt *gitlab.ListGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, opt)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 []*gitlab.Group
	if rf, ok := ret.Get(0).(func(*gitlab.ListGroupsOptions, ...gitlab.RequestOptionFunc) []*gitlab.Group); ok {
		r0 = rf(opt, options...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*gitlab.Group)
		}
	}

	var r1 *gitlab.Response
	if rf, ok := ret.Get(1).(func(*gitlab.ListGroupsOptions, ...gitlab.RequestOptionFunc) *gitlab.Response); ok {
		r1 = rf(opt, options...)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*gitlab.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*gitlab.ListGroupsOptions, ...gitlab.RequestOptionFunc) error); ok {
		r2 = rf(opt, options...)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
