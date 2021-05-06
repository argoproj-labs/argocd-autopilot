package fs

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_fs_Exists(t *testing.T) {
	tests := map[string]struct {
		fs       billy.Filesystem
		beforeFn func(fs FS)
		path     string
		want     bool
		wantErr  bool
	}{
		"Should exist": {
			fs:   memfs.New(),
			path: "/foo/noam/bar",
			beforeFn: func(fs FS) {
				f, err := fs.Create("/foo/noam/bar")
				util.Die(err)
				defer f.Close()
			},
			want:    true,
			wantErr: false,
		},
		"Should not exist": {
			fs:       memfs.New(),
			path:     "/foo/noam/bar",
			beforeFn: func(fs FS) {},
			want:     false,
			wantErr:  false,
		},
		"Should throw error": {
			fs:   &mocks.FS{},
			path: "invalid file path",
			beforeFn: func(fs FS) {
				f := fs.(*fsimpl)
				m := f.Filesystem.(*mocks.FS)
				m.On("Stat", mock.Anything).Return(nil, fmt.Errorf("error"))
			},
			want:    false,
			wantErr: true,
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			fs := &fsimpl{tt.fs}
			tt.beforeFn(fs)
			got, err := fs.Exists(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("fs.Exists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("fs.Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	tests := map[string]struct {
		bfs      billy.Filesystem
		beforeFn func(fs billy.Filesystem) FS
	}{
		"should create FS": {
			bfs: memfs.New(),
			beforeFn: func(bfs billy.Filesystem) FS {
				return &fsimpl{bfs}
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			want := tt.beforeFn(tt.bfs)
			if got := Create(tt.bfs); !reflect.DeepEqual(got, want) {
				t.Errorf("Create() = %v, want %v", got, want)
			}
		})
	}
}

func Test_fsimpl_CheckExistsOrWrite(t *testing.T) {
	type args struct {
		path string
		data []byte
	}
	tests := map[string]struct {
		args     args
		want     bool
		wantErr  string
		beforeFn func(m *mocks.FS)
		assertFn func(t *testing.T, mockedFile *mocks.File)
	}{
		"should exists": {
			args: args{path: "/usr/bar", data: []byte{}},
			want: true,
			beforeFn: func(m *mocks.FS) {
				m.On("Stat", "/usr/bar").Return(nil, nil)
			},
		},
		"should error on fail check": {
			args:    args{path: "/usr/bar", data: []byte{}},
			wantErr: "failed to check if file exists on repo at '/usr/bar': some error",
			beforeFn: func(m *mocks.FS) {
				m.On("Stat", "/usr/bar").Return(nil, fmt.Errorf("some error"))
			},
		},
		"should write to file if not exists and write sucsseded": {
			args: args{path: "/usr/bar", data: []byte{}},
			want: false,
			beforeFn: func(m *mocks.FS) {
				mfile := &mocks.File{}
				mfile.On("Write", mock.AnythingOfType("[]uint8")).Return(1, nil)
				mfile.On("Close").Return(nil)
				m.On("Stat", "/usr/bar").Return(nil, os.ErrNotExist)
				m.On("OpenFile", "/usr/bar", mock.AnythingOfType("int"), mock.AnythingOfType("FileMode")).Return(mfile, nil)
			},
		},
		"should fail if WriteFile failed": {
			args:    args{path: "/usr/bar", data: []byte{}},
			want:    false,
			wantErr: "failed to create file at '/usr/bar': " + os.ErrPermission.Error(),
			beforeFn: func(m *mocks.FS) {
				m.On("Stat", "/usr/bar").Return(nil, os.ErrNotExist)
				m.On("OpenFile", "/usr/bar", mock.AnythingOfType("int"), mock.AnythingOfType("FileMode")).Return(nil, os.ErrPermission)
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			mockedFS := &mocks.FS{}
			tt.beforeFn(mockedFS)
			fs := Create(mockedFS)
			got, err := fs.CheckExistsOrWrite(tt.args.path, tt.args.data)

			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("prepare() error = %v", err)
				}

				return
			}

			if got != tt.want {
				t.Errorf("fsimpl.CheckExistsOrWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fsimpl_ExistsOrDie(t *testing.T) {
	tests := map[string]struct {
		path     string
		want     bool
		wantErr  bool
		beforeFn func(m *mocks.FS)
	}{
		"should exists if path exists": {
			path:    "/usr/bar",
			want:    true,
			wantErr: false,
			beforeFn: func(m *mocks.FS) {
				m.On("Stat", mock.Anything).Return(nil, nil)
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			mock := &mocks.FS{}
			fs := Create(mock)
			tt.beforeFn(mock)
			if got := fs.ExistsOrDie(tt.path); got != tt.want {
				t.Errorf("fsimpl.ExistsOrDie() = %v, want %v", got, tt.want)
			}
		})
	}
}
