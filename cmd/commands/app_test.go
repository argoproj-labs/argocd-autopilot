package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/application"
	appmocks "github.com/argoproj/argocd-autopilot/pkg/application/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	fsmocks "github.com/argoproj/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/kube"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/ghodss/yaml"
	"github.com/go-git/go-billy/v5/memfs"
	osfs "github.com/go-git/go-billy/v5/osfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	kusttypes "sigs.k8s.io/kustomize/api/types"
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
				mfs := &fsmocks.FS{}
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

	getAppMock := func() *appmocks.Application {
		app := &appmocks.Application{}
		app.On("Name").Return("foo")
		app.On("Config").Return(&application.Config{})
		app.On("Namespace").Return(kube.GenerateNamespace("foo"))
		app.On("Manifests").Return(nil)
		app.On("Base").Return(&kusttypes.Kustomization{
			TypeMeta: kusttypes.TypeMeta{
				APIVersion: kusttypes.KustomizationVersion,
				Kind:       kusttypes.KustomizationKind,
			},
			Resources: []string{"foo"},
		})
		app.On("Overlay").Return(&kusttypes.Kustomization{
			TypeMeta: kusttypes.TypeMeta{
				APIVersion: kusttypes.KustomizationVersion,
				Kind:       kusttypes.KustomizationKind,
			},
			Resources: []string{"foo"},
		})
		return app
	}

	tests := map[string]struct {
		projectName string
		beforeFn    func(*testing.T) (fs.FS, application.Application, string)
		assertFn    func(*testing.T, fs.FS, application.Application, error)
	}{
		"New application": {
			beforeFn: func(t *testing.T) (fs.FS, application.Application, string) {
				app := getAppMock()
				root, err := ioutil.TempDir("", "test")
				assert.NoError(t, err)
				repofs := fs.Create(osfs.New(root))

				return repofs, app, "fooproj"
			},
			assertFn: func(t *testing.T, repofs fs.FS, a application.Application, ret error) {
				defer os.RemoveAll(repofs.Root()) // remove temp dir
				assert.NoError(t, ret)
				assert.DirExists(t, repofs.Join(repofs.Root(), store.Default.KustomizeDir), "kustomization dir should exist")
				assert.FileExists(t, repofs.Join(repofs.Root(), store.Default.KustomizeDir, "foo", "base", "kustomization.yaml"))
				assert.FileExists(t, repofs.Join(repofs.Root(), store.Default.KustomizeDir, "foo", "overlays", "fooproj", "kustomization.yaml"))
				assert.FileExists(t, repofs.Join(repofs.Root(), store.Default.KustomizeDir, "foo", "overlays", "fooproj", "namespace.yaml"))
				assert.FileExists(t, repofs.Join(repofs.Root(), store.Default.KustomizeDir, "foo", "overlays", "fooproj", "config.json"))
			},
		},
		"App base collision": {
			beforeFn: func(t *testing.T) (fs.FS, application.Application, string) {
				app := getAppMock()
				root, err := ioutil.TempDir("", "test")
				assert.NoError(t, err)
				repofs := fs.Create(osfs.New(root))
				// change the original base to make collision with the new one
				orgBase := &kusttypes.Kustomization{
					TypeMeta: kusttypes.TypeMeta{
						APIVersion: kusttypes.KustomizationVersion,
						Kind:       kusttypes.KustomizationKind,
					},
					Resources: []string{"bar"}, // different resources
				}
				orgBaseYAML, err := yaml.Marshal(orgBase)
				assert.NoError(t, err)
				_, err = repofs.WriteFile(repofs.Join(store.Default.KustomizeDir, "foo", "base", "kustomization.yaml"), orgBaseYAML)
				assert.NoError(t, err)
				_, err = repofs.WriteFile(repofs.Join(store.Default.KustomizeDir, "foo", "overlays", "fooproj", "kustomization.yaml"), []byte(""))
				assert.NoError(t, err)

				return repofs, app, "fooproj"
			},
			assertFn: func(t *testing.T, repofs fs.FS, a application.Application, ret error) {
				defer os.RemoveAll(repofs.Root()) // remove temp dir
				assert.ErrorIs(t, ret, ErrAppCollisionWithExistingBase)
			},
		},
		"Same app base overlay already exists": {
			beforeFn: func(t *testing.T) (fs.FS, application.Application, string) {
				app := getAppMock()
				root, err := ioutil.TempDir("", "test")
				assert.NoError(t, err)
				repofs := fs.Create(osfs.New(root))
				sameBase, err := yaml.Marshal(app.Base())
				assert.NoError(t, err)
				_, err = repofs.WriteFile(repofs.Join(store.Default.KustomizeDir, "foo", "base", "kustomization.yaml"), sameBase)
				assert.NoError(t, err)
				_, err = repofs.WriteFile(repofs.Join(store.Default.KustomizeDir, "foo", "overlays", "fooproj", "kustomization.yaml"), []byte(""))
				assert.NoError(t, err)

				return repofs, app, "fooproj"
			},
			assertFn: func(t *testing.T, repofs fs.FS, a application.Application, ret error) {
				defer os.RemoveAll(repofs.Root()) // remove temp dir
				assert.ErrorIs(t, ret, ErrAppAlreadyInstalledOnProject)
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			repofs, app, proj := tt.beforeFn(t)
			err := createApplicationFiles(repofs, app, proj)
			tt.assertFn(t, repofs, app, err)
		})
	}
}
