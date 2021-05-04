package commands

import (
	"context"
	_ "embed"
	"fmt"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	gitmocks "github.com/argoproj/argocd-autopilot/pkg/git/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRunRepoCreate(t *testing.T) {
	tests := map[string]struct {
		opts     *RepoCreateOptions
		preFn    func(*gitmocks.Provider)
		assertFn func(t *testing.T, mp *gitmocks.Provider, opts *RepoCreateOptions, ret error)
	}{
		"Invalid provider": {
			opts: &RepoCreateOptions{
				Provider: "foobar",
			},
			assertFn: func(t *testing.T, mp *gitmocks.Provider, opts *RepoCreateOptions, ret error) {
				assert.ErrorIs(t, ret, git.ErrProviderNotSupported)
			},
		},
		"Should call CreateRepository": {
			opts: &RepoCreateOptions{
				Provider: "github",
				Owner:    "foo",
				Repo:     "bar",
				Token:    "test",
				Public:   false,
				Host:     "",
			},
			preFn: func(mp *gitmocks.Provider) {
				mp.On("CreateRepository", mock.Anything, mock.Anything).Return("", nil)
			},
			assertFn: func(t *testing.T, mp *gitmocks.Provider, opts *RepoCreateOptions, ret error) {
				mp.AssertCalled(t, "CreateRepository", mock.Anything, mock.Anything)
				o := mp.Calls[0].Arguments[1].(*git.CreateRepoOptions)
				assert.NotNil(t, o)
				assert.Equal(t, opts.Public, !o.Private)
			},
		},
		"Should fail to CreateRepository": {
			opts: &RepoCreateOptions{
				Provider: "github",
				Owner:    "foo",
				Repo:     "bar",
				Token:    "test",
				Public:   false,
				Host:     "",
			},
			preFn: func(mp *gitmocks.Provider) {
				mp.On("CreateRepository", mock.Anything, mock.Anything).Return("", fmt.Errorf("error"))
			},
			assertFn: func(t *testing.T, mp *gitmocks.Provider, opts *RepoCreateOptions, ret error) {
				mp.AssertCalled(t, "CreateRepository", mock.Anything, mock.Anything)
				assert.EqualError(t, ret, "error")
			},
		},
	}

	orgGetProvider := getGitProvider
	for tname, tt := range tests {
		defer func() { getGitProvider = orgGetProvider }()
		mp := &gitmocks.Provider{}
		if tt.preFn != nil {
			tt.preFn(mp)
			getGitProvider = func(opts *git.ProviderOptions) (git.Provider, error) {
				return mp, nil
			}
		}

		t.Run(tname, func(t *testing.T) {
			tt.assertFn(t, mp, tt.opts, RunRepoCreate(context.Background(), tt.opts))
		})
	}
}

func Test_setBootstrapOptsDefaults(t *testing.T) {
	tests := map[string]struct {
		opts     *RepoBootstrapOptions
		preFn    func()
		assertFn func(t *testing.T, opts *RepoBootstrapOptions, ret error)
	}{
		"Bad installation mode": {
			opts: &RepoBootstrapOptions{
				InstallationMode: "foo",
			},
			assertFn: func(t *testing.T, opts *RepoBootstrapOptions, ret error) {
				assert.EqualError(t, ret, "unknown installation mode: foo")
			},
		},
		"Basic": {
			opts: &RepoBootstrapOptions{},
			preFn: func() {
				currentKubeContext = func() (string, error) {
					return "fooctx", nil
				}
			},
			assertFn: func(t *testing.T, opts *RepoBootstrapOptions, ret error) {
				assert.NoError(t, ret)
				assert.Equal(t, "argocd", opts.Namespace)
				assert.Equal(t, false, opts.Namespaced)
				assert.Equal(t, "manifests", opts.AppSpecifier)
				assert.Equal(t, "fooctx", opts.KubeContext)
			},
		},
		"With App specifier": {
			opts: &RepoBootstrapOptions{
				AppSpecifier: "https://github.com/foo/bar",
				KubeContext:  "fooctx",
			},
			assertFn: func(t *testing.T, opts *RepoBootstrapOptions, ret error) {
				assert.NoError(t, ret)
				assert.Equal(t, "argocd", opts.Namespace)
				assert.Equal(t, false, opts.Namespaced)
				assert.Equal(t, installationModeNormal, opts.InstallationMode)
				assert.Equal(t, "https://github.com/foo/bar", opts.AppSpecifier)
				assert.Equal(t, "fooctx", opts.KubeContext)
			},
		},
		"Namespaced": {
			opts: &RepoBootstrapOptions{
				InstallationMode: installationModeFlat,
				KubeContext:      "fooctx",
				Namespaced:       true,
				Namespace:        "bar",
			},
			assertFn: func(t *testing.T, opts *RepoBootstrapOptions, ret error) {
				assert.NoError(t, ret)
				assert.Equal(t, "bar", opts.Namespace)
				assert.Equal(t, true, opts.Namespaced)
				assert.Equal(t, installationModeFlat, opts.InstallationMode)
				assert.Equal(t, "manifests/namespace-install", opts.AppSpecifier)
				assert.Equal(t, "fooctx", opts.KubeContext)
			},
		},
	}

	orgCurrentKubeContext := currentKubeContext
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			if tt.preFn != nil {
				tt.preFn()
				defer func() { currentKubeContext = orgCurrentKubeContext }()
			}

			ret, err := setBootstrapOptsDefaults(*tt.opts)
			tt.assertFn(t, ret, err)
		})
	}
}

func Test_validateRepo(t *testing.T) {
	tests := map[string]struct {
		preFn    func(t *testing.T, repofs fs.FS)
		assertFn func(t *testing.T, repofs fs.FS, ret error)
	}{
		"Bootstrap exists": {
			preFn: func(t *testing.T, repofs fs.FS) {
				_, err := repofs.WriteFile(store.Default.BootsrtrapDir, []byte{})
				assert.NoError(t, err)
			},
			assertFn: func(t *testing.T, repofs fs.FS, ret error) {
				assert.EqualError(t, ret, fmt.Sprintf("folder %[1]s already exist in: /%[1]s", store.Default.BootsrtrapDir))
			},
		},
		"Projects exists": {
			preFn: func(t *testing.T, repofs fs.FS) {
				_, err := repofs.WriteFile(store.Default.ProjectsDir, []byte{})
				assert.NoError(t, err)
			},
			assertFn: func(t *testing.T, repofs fs.FS, ret error) {
				assert.EqualError(t, ret, fmt.Sprintf("folder %[1]s already exist in: /%[1]s", store.Default.ProjectsDir))
			},
		},
		"Valid": {
			assertFn: func(t *testing.T, repofs fs.FS, ret error) {
				assert.NoError(t, ret)
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			repofs := fs.Create(memfs.New())
			if tt.preFn != nil {
				tt.preFn(t, repofs)
			}

			tt.assertFn(t, repofs, validateRepo(repofs))
		})
	}
}
