// Code generated by mockery (devel). DO NOT EDIT.

package mocks

import (
	context "context"

	github "github.com/google/go-github/v34/github"

	mock "github.com/stretchr/testify/mock"
)

// Users is an autogenerated mock type for the Users type
type Users struct {
	mock.Mock
}

// AcceptInvitation provides a mock function with given fields: _a0, _a1
func (_m *Users) AcceptInvitation(_a0 context.Context, _a1 int64) (*github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, int64) *github.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// AddEmails provides a mock function with given fields: _a0, _a1
func (_m *Users) AddEmails(_a0 context.Context, _a1 []string) ([]*github.UserEmail, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []*github.UserEmail
	if rf, ok := ret.Get(0).(func(context.Context, []string) []*github.UserEmail); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*github.UserEmail)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, []string) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, []string) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// BlockUser provides a mock function with given fields: _a0, _a1
func (_m *Users) BlockUser(_a0 context.Context, _a1 string) (*github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, string) *github.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateGPGKey provides a mock function with given fields: _a0, _a1
func (_m *Users) CreateGPGKey(_a0 context.Context, _a1 string) (*github.GPGKey, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.GPGKey
	if rf, ok := ret.Get(0).(func(context.Context, string) *github.GPGKey); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.GPGKey)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// CreateKey provides a mock function with given fields: _a0, _a1
func (_m *Users) CreateKey(_a0 context.Context, _a1 *github.Key) (*github.Key, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Key
	if rf, ok := ret.Get(0).(func(context.Context, *github.Key) *github.Key); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Key)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, *github.Key) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, *github.Key) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// CreateProject provides a mock function with given fields: _a0, _a1
func (_m *Users) CreateProject(_a0 context.Context, _a1 *github.CreateUserProjectOptions) (*github.Project, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Project
	if rf, ok := ret.Get(0).(func(context.Context, *github.CreateUserProjectOptions) *github.Project); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Project)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, *github.CreateUserProjectOptions) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, *github.CreateUserProjectOptions) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// DeclineInvitation provides a mock function with given fields: _a0, _a1
func (_m *Users) DeclineInvitation(_a0 context.Context, _a1 int64) (*github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, int64) *github.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteEmails provides a mock function with given fields: _a0, _a1
func (_m *Users) DeleteEmails(_a0 context.Context, _a1 []string) (*github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, []string) *github.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, []string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteGPGKey provides a mock function with given fields: _a0, _a1
func (_m *Users) DeleteGPGKey(_a0 context.Context, _a1 int64) (*github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, int64) *github.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteKey provides a mock function with given fields: _a0, _a1
func (_m *Users) DeleteKey(_a0 context.Context, _a1 int64) (*github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, int64) *github.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DemoteSiteAdmin provides a mock function with given fields: _a0, _a1
func (_m *Users) DemoteSiteAdmin(_a0 context.Context, _a1 string) (*github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, string) *github.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Edit provides a mock function with given fields: _a0, _a1
func (_m *Users) Edit(_a0 context.Context, _a1 *github.User) (*github.User, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.User
	if rf, ok := ret.Get(0).(func(context.Context, *github.User) *github.User); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.User)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, *github.User) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, *github.User) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// Follow provides a mock function with given fields: _a0, _a1
func (_m *Users) Follow(_a0 context.Context, _a1 string) (*github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, string) *github.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Get provides a mock function with given fields: _a0, _a1
func (_m *Users) Get(_a0 context.Context, _a1 string) (*github.User, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.User
	if rf, ok := ret.Get(0).(func(context.Context, string) *github.User); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.User)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetByID provides a mock function with given fields: _a0, _a1
func (_m *Users) GetByID(_a0 context.Context, _a1 int64) (*github.User, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.User
	if rf, ok := ret.Get(0).(func(context.Context, int64) *github.User); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.User)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, int64) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, int64) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetGPGKey provides a mock function with given fields: _a0, _a1
func (_m *Users) GetGPGKey(_a0 context.Context, _a1 int64) (*github.GPGKey, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.GPGKey
	if rf, ok := ret.Get(0).(func(context.Context, int64) *github.GPGKey); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.GPGKey)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, int64) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, int64) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetHovercard provides a mock function with given fields: _a0, _a1, _a2
func (_m *Users) GetHovercard(_a0 context.Context, _a1 string, _a2 *github.HovercardOptions) (*github.Hovercard, *github.Response, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *github.Hovercard
	if rf, ok := ret.Get(0).(func(context.Context, string, *github.HovercardOptions) *github.Hovercard); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Hovercard)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string, *github.HovercardOptions) *github.Response); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, *github.HovercardOptions) error); ok {
		r2 = rf(_a0, _a1, _a2)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetKey provides a mock function with given fields: _a0, _a1
func (_m *Users) GetKey(_a0 context.Context, _a1 int64) (*github.Key, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Key
	if rf, ok := ret.Get(0).(func(context.Context, int64) *github.Key); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Key)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, int64) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, int64) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// IsBlocked provides a mock function with given fields: _a0, _a1
func (_m *Users) IsBlocked(_a0 context.Context, _a1 string) (bool, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string) bool); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// IsFollowing provides a mock function with given fields: _a0, _a1, _a2
func (_m *Users) IsFollowing(_a0 context.Context, _a1 string, _a2 string) (bool, *github.Response, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string, string) bool); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string, string) *github.Response); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, string) error); ok {
		r2 = rf(_a0, _a1, _a2)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ListAll provides a mock function with given fields: _a0, _a1
func (_m *Users) ListAll(_a0 context.Context, _a1 *github.UserListOptions) ([]*github.User, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []*github.User
	if rf, ok := ret.Get(0).(func(context.Context, *github.UserListOptions) []*github.User); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*github.User)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, *github.UserListOptions) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, *github.UserListOptions) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ListBlockedUsers provides a mock function with given fields: _a0, _a1
func (_m *Users) ListBlockedUsers(_a0 context.Context, _a1 *github.ListOptions) ([]*github.User, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []*github.User
	if rf, ok := ret.Get(0).(func(context.Context, *github.ListOptions) []*github.User); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*github.User)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, *github.ListOptions) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, *github.ListOptions) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ListEmails provides a mock function with given fields: _a0, _a1
func (_m *Users) ListEmails(_a0 context.Context, _a1 *github.ListOptions) ([]*github.UserEmail, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []*github.UserEmail
	if rf, ok := ret.Get(0).(func(context.Context, *github.ListOptions) []*github.UserEmail); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*github.UserEmail)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, *github.ListOptions) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, *github.ListOptions) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ListFollowers provides a mock function with given fields: _a0, _a1, _a2
func (_m *Users) ListFollowers(_a0 context.Context, _a1 string, _a2 *github.ListOptions) ([]*github.User, *github.Response, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []*github.User
	if rf, ok := ret.Get(0).(func(context.Context, string, *github.ListOptions) []*github.User); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*github.User)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string, *github.ListOptions) *github.Response); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, *github.ListOptions) error); ok {
		r2 = rf(_a0, _a1, _a2)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ListFollowing provides a mock function with given fields: _a0, _a1, _a2
func (_m *Users) ListFollowing(_a0 context.Context, _a1 string, _a2 *github.ListOptions) ([]*github.User, *github.Response, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []*github.User
	if rf, ok := ret.Get(0).(func(context.Context, string, *github.ListOptions) []*github.User); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*github.User)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string, *github.ListOptions) *github.Response); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, *github.ListOptions) error); ok {
		r2 = rf(_a0, _a1, _a2)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ListGPGKeys provides a mock function with given fields: _a0, _a1, _a2
func (_m *Users) ListGPGKeys(_a0 context.Context, _a1 string, _a2 *github.ListOptions) ([]*github.GPGKey, *github.Response, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []*github.GPGKey
	if rf, ok := ret.Get(0).(func(context.Context, string, *github.ListOptions) []*github.GPGKey); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*github.GPGKey)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string, *github.ListOptions) *github.Response); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, *github.ListOptions) error); ok {
		r2 = rf(_a0, _a1, _a2)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ListInvitations provides a mock function with given fields: _a0, _a1
func (_m *Users) ListInvitations(_a0 context.Context, _a1 *github.ListOptions) ([]*github.RepositoryInvitation, *github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []*github.RepositoryInvitation
	if rf, ok := ret.Get(0).(func(context.Context, *github.ListOptions) []*github.RepositoryInvitation); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*github.RepositoryInvitation)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, *github.ListOptions) *github.Response); ok {
		r1 = rf(_a0, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, *github.ListOptions) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ListKeys provides a mock function with given fields: _a0, _a1, _a2
func (_m *Users) ListKeys(_a0 context.Context, _a1 string, _a2 *github.ListOptions) ([]*github.Key, *github.Response, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []*github.Key
	if rf, ok := ret.Get(0).(func(context.Context, string, *github.ListOptions) []*github.Key); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*github.Key)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string, *github.ListOptions) *github.Response); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, *github.ListOptions) error); ok {
		r2 = rf(_a0, _a1, _a2)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ListProjects provides a mock function with given fields: _a0, _a1, _a2
func (_m *Users) ListProjects(_a0 context.Context, _a1 string, _a2 *github.ProjectListOptions) ([]*github.Project, *github.Response, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []*github.Project
	if rf, ok := ret.Get(0).(func(context.Context, string, *github.ProjectListOptions) []*github.Project); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*github.Project)
		}
	}

	var r1 *github.Response
	if rf, ok := ret.Get(1).(func(context.Context, string, *github.ProjectListOptions) *github.Response); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*github.Response)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, *github.ProjectListOptions) error); ok {
		r2 = rf(_a0, _a1, _a2)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// PromoteSiteAdmin provides a mock function with given fields: _a0, _a1
func (_m *Users) PromoteSiteAdmin(_a0 context.Context, _a1 string) (*github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, string) *github.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Suspend provides a mock function with given fields: _a0, _a1, _a2
func (_m *Users) Suspend(_a0 context.Context, _a1 string, _a2 *github.UserSuspendOptions) (*github.Response, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, string, *github.UserSuspendOptions) *github.Response); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *github.UserSuspendOptions) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UnblockUser provides a mock function with given fields: _a0, _a1
func (_m *Users) UnblockUser(_a0 context.Context, _a1 string) (*github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, string) *github.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Unfollow provides a mock function with given fields: _a0, _a1
func (_m *Users) Unfollow(_a0 context.Context, _a1 string) (*github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, string) *github.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Unsuspend provides a mock function with given fields: _a0, _a1
func (_m *Users) Unsuspend(_a0 context.Context, _a1 string) (*github.Response, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *github.Response
	if rf, ok := ret.Get(0).(func(context.Context, string) *github.Response); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*github.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
