package commands

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/application"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_getCommitMsg(t *testing.T) {
	tests := map[string]struct {
		opts     *AppCreateOptions
		assertFn func(t *testing.T, res string)
	}{
		"On root": {
			opts: &AppCreateOptions{
				CloneOptions: &git.CloneOptions{
					RepoRoot: "",
				},
				AppOpts: &application.CreateOptions{
					AppName: "foo",
				},
				ProjectName: "bar",
			},
			assertFn: func(t *testing.T, res string) {
				assert.Contains(t, res, "installed app 'foo' on project 'bar'")
				assert.NotContains(t, res, "installation-path")
			},
		},
		"On installation path": {
			opts: &AppCreateOptions{
				CloneOptions: &git.CloneOptions{
					RepoRoot: "foo/bar",
				},
				AppOpts: &application.CreateOptions{
					AppName: "foo",
				},
				ProjectName: "bar",
			},
			assertFn: func(t *testing.T, res string) {
				assert.Contains(t, res, "installed app 'foo' on project 'bar'")
				assert.Contains(t, res, "installation-path: 'foo/bar'")
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			got := getCommitMsg(tt.opts)
			tt.assertFn(t, got)
		})
	}
}

func Test_writeApplicationFile(t *testing.T) {
	type args struct {
		root string
		path string
		name string
		data []byte
	}
	tests := map[string]struct {
		args     args
		assertFn func(t *testing.T, repofs fs.FS, exists bool, err error)
		beforeFn func(repofs fs.FS) fs.FS
	}{
		"On Root": {
			args: args{
				path: "foo/bar",
				name: "test",
				data: []byte("data"),
			},
			assertFn: func(t *testing.T, repofs fs.FS, exists bool, ret error) {
				assert.NoError(t, ret)

				f, err := repofs.Open("/foo/bar")
				assert.NoError(t, err)
				d, err := ioutil.ReadAll(f)
				assert.NoError(t, err)

				assert.Equal(t, d, []byte("data"))
				assert.False(t, exists)
			},
		},
		"With Chroot": {
			args: args{
				root: "someroot",
				path: "foo/bar",
				name: "test",
				data: []byte("data2"),
			},
			assertFn: func(t *testing.T, repofs fs.FS, exists bool, ret error) {
				assert.NoError(t, ret)

				assert.Equal(t, "/someroot", repofs.Root())
				f, err := repofs.Open("/foo/bar")
				assert.NoError(t, err)
				d, err := ioutil.ReadAll(f)
				assert.NoError(t, err)

				assert.Equal(t, d, []byte("data2"))
				assert.False(t, exists)
			},
		},
		"File exists": {
			args: args{
				path: "foo/bar",
				name: "test",
				data: []byte("data2"),
			},
			beforeFn: func(repofs fs.FS) fs.FS {
				_, _ = repofs.WriteFile("/foo/bar", []byte("data"))
				return repofs
			},
			assertFn: func(t *testing.T, repofs fs.FS, exists bool, ret error) {
				assert.NoError(t, ret)
				assert.True(t, exists)
			},
		},
		"Write error": {
			args: args{
				path: "foo/bar",
				name: "test",
				data: []byte("data2"),
			},
			beforeFn: func(repofs fs.FS) fs.FS {
				mfs := &mocks.FS{}
				mfs.On("CheckExistsOrWrite", mock.Anything, mock.Anything).Return(false, fmt.Errorf("error"))
				mfs.On("Root").Return("/")
				mfs.On("Join", mock.Anything, mock.Anything).Return("/foo/bar")

				return mfs
			},
			assertFn: func(t *testing.T, repofs fs.FS, exists bool, ret error) {
				assert.Error(t, ret)
				assert.EqualError(t, ret, "failed to create 'test' file at '/foo/bar': error")
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			repofs := fs.Create(memfs.New())
			if tt.args.root != "" {
				bfs, _ := repofs.Chroot(tt.args.root)
				repofs = fs.Create(bfs)
			}
			if tt.beforeFn != nil {
				repofs = tt.beforeFn(repofs)
			}
			got, err := writeApplicationFile(repofs, tt.args.path, tt.args.name, tt.args.data)
			tt.assertFn(t, repofs, got, err)
		})
	}
}

func Test_createApplicationFiles(t *testing.T) {
	type args struct {
		repoFS      fs.FS
		app         application.Application
		projectName string
	}
	tests := map[string]struct {
		args    args
		wantErr bool
	}{
		"": {},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			if err := createApplicationFiles(tt.args.repoFS, tt.args.app, tt.args.projectName); (err != nil) != tt.wantErr {
				t.Errorf("createApplicationFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
