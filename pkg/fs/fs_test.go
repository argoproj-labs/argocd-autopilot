package fs

import (
	"fmt"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/stretchr/testify/mock"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
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
