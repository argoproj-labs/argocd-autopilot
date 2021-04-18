package fs

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
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
		wantErr  bool
		beforeFn func(fs FS)
		fs       billy.Filesystem
	}{
		"should exists": {
			args:    args{path: "/usr/bar", data: []byte{}},
			want:    true,
			wantErr: false,
			beforeFn: func(fs FS) {
				f := fs.(*fsimpl)
				m := f.Filesystem.(*mocks.FS)
				m.On("Stat", mock.Anything).Return(nil, nil)
			},
			fs: &mocks.FS{},
		},
		"should error on fail check": {
			args:    args{path: "/usr/bar", data: []byte{}},
			wantErr: true,
			beforeFn: func(fs FS) {
				f := fs.(*fsimpl)
				m := f.Filesystem.(*mocks.FS)
				m.On("Stat", mock.Anything).Return(nil, fmt.Errorf("error"))
			},
			fs: &mocks.FS{},
		},
		// "should write to file if not exists": {
		// 	args:    args{path: "/usr/bar", data: []byte{}},
		// 	wantErr: false,
		// 	beforeFn: func(fs FS) {
		// 		f := fs.(*fsimpl)
		// 		m := f.Filesystem.(*mocks.FS)
		// 		m.On("Stat", mock.Anything).Return(nil, os.ErrNotExist)
		// 		m.On("Create", mock.Anything).Return()
		// 	},
		// 	fs: &mocks.FS{},
		// },
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			fs := &fsimpl{tt.fs}
			tt.beforeFn(fs)
			got, err := fs.CheckExistsOrWrite(tt.args.path, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("fsimpl.CheckExistsOrWrite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("fsimpl.CheckExistsOrWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fsimpl_ExistsOrDie(t *testing.T) {

	type args struct {
		path string
	}
	tests := map[string]struct {
		args     args
		want     bool
		wantErr  bool
		beforeFn func(fs FS)
		fs       billy.Filesystem
	}{
		"should exists if path exists": {
			args:    args{path: "/usr/bar"},
			want:    true,
			wantErr: false,
			beforeFn: func(fs FS) {
				f := fs.(*fsimpl)
				m := f.Filesystem.(*mocks.FS)
				m.On("Stat", mock.Anything).Return(nil, nil)
			},
			fs: &mocks.FS{},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			fs := &fsimpl{tt.fs}
			tt.beforeFn(fs)
			if got := fs.ExistsOrDie(tt.args.path); got != tt.want {
				t.Errorf("fsimpl.ExistsOrDie() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fsimpl_ChrootOrDie(t *testing.T) {

	type args struct {
		newRoot string
	}
	tests := map[string]struct {
		args   args
		wantErr bool
		beforeFn func(fs FS)
		fs FS
	}{
			"should exists if path exists": {
				args:  args{newRoot: "root"},
				wantErr: false,
				beforeFn: func(fs FS) {
					f := fs.(*fsimpl)
					m := f.Filesystem.(*mocks.FS)
					m.On("Chroot", mock.Anything).Return(nil, nil)
				},
				fs: &mocks.FS{},
			},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			fs := &fsimpl{
				tt.fs,
			}
			tt.beforeFn(fs)
			fs.ChrootOrDie(tt.args.newRoot)
		})
	}
}
