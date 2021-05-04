package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/argocd"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	gitmocks "github.com/argoproj/argocd-autopilot/pkg/git/mocks"
	kubemocks "github.com/argoproj/argocd-autopilot/pkg/kube/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/store"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/ghodss/yaml"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	kusttypes "sigs.k8s.io/kustomize/api/types"
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

func Test_buildBootstrapManifests(t *testing.T) {
	type args struct {
		namespace    string
		appSpecifier string
		cloneOpts    *git.CloneOptions
	}
	tests := map[string]struct {
		args     args
		preFn    func()
		assertFn func(t *testing.T, b *bootstrapManifests, ret error)
	}{
		"Basic": {
			args: args{
				namespace:    "foo",
				appSpecifier: "bar",
				cloneOpts: &git.CloneOptions{
					Revision: "main",
					URL:      "https://github.com/foo/bar",
					RepoRoot: "/installation1",
					Auth:     git.Auth{Password: "test"},
				},
			},
			preFn: func() {
				runKustomizeBuild = func(k *kusttypes.Kustomization) ([]byte, error) {
					return []byte("test"), nil
				}
			},
			assertFn: func(t *testing.T, b *bootstrapManifests, ret error) {
				assert.NoError(t, ret)
				assert.Equal(t, []byte("test"), b.applyManifests)

				argocdApp := &argocdv1alpha1.Application{}
				assert.NoError(t, yaml.Unmarshal(b.argocdApp, argocdApp))
				assert.Equal(t, argocdApp.Spec.Source.RepoURL, "https://github.com/foo/bar")
				assert.Equal(t, argocdApp.Spec.Source.Path, filepath.Join(
					"/installation1",
					store.Default.BootsrtrapDir,
					store.Default.ArgoCDName,
				))
				assert.Equal(t, argocdApp.Spec.Source.TargetRevision, "main")
				assert.Equal(t, 0, len(argocdApp.ObjectMeta.Finalizers))
				assert.Equal(t, argocdApp.Spec.Destination.Namespace, "foo")
				assert.Equal(t, argocdApp.Spec.Destination.Server, store.Default.DestServer)

				bootstrapApp := &argocdv1alpha1.Application{}
				assert.NoError(t, yaml.Unmarshal(b.bootstrapApp, bootstrapApp))
				assert.Equal(t, bootstrapApp.Spec.Source.RepoURL, "https://github.com/foo/bar")
				assert.Equal(t, bootstrapApp.Spec.Source.Path, filepath.Join(
					"/installation1",
					store.Default.BootsrtrapDir,
				))
				assert.Equal(t, bootstrapApp.Spec.Source.TargetRevision, "main")
				assert.NotEqual(t, 0, len(bootstrapApp.ObjectMeta.Finalizers))
				assert.Equal(t, bootstrapApp.Spec.Destination.Namespace, "foo")
				assert.Equal(t, bootstrapApp.Spec.Destination.Server, store.Default.DestServer)

				rootApp := &argocdv1alpha1.Application{}
				assert.NoError(t, yaml.Unmarshal(b.rootApp, rootApp))
				assert.Equal(t, rootApp.Spec.Source.RepoURL, "https://github.com/foo/bar")
				assert.Equal(t, rootApp.Spec.Source.Path, filepath.Join(
					"/installation1",
					store.Default.ProjectsDir,
				))
				assert.Equal(t, rootApp.Spec.Source.TargetRevision, "main")
				assert.NotEqual(t, 0, len(rootApp.ObjectMeta.Finalizers))
				assert.Equal(t, rootApp.Spec.Destination.Namespace, "foo")
				assert.Equal(t, rootApp.Spec.Destination.Server, store.Default.DestServer)

				ns := &v1.Namespace{}
				assert.NoError(t, yaml.Unmarshal(b.namespace, ns))
				assert.Equal(t, ns.ObjectMeta.Name, "foo")

				creds := &v1.Secret{}
				assert.NoError(t, yaml.Unmarshal(b.repoCreds, &creds))
				assert.Equal(t, creds.ObjectMeta.Name, store.Default.RepoCredsSecretName)
				assert.Equal(t, creds.ObjectMeta.Namespace, "foo")
				assert.Equal(t, creds.Data["git_token"], []byte("test"))
				assert.Equal(t, creds.Data["git_username"], []byte(store.Default.GitUsername))
			},
		},
	}

	orgRunKustomizeBuild := runKustomizeBuild

	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			if tt.preFn != nil {
				tt.preFn()
				defer func() { runKustomizeBuild = orgRunKustomizeBuild }()
			}
			b, ret := buildBootstrapManifests(
				tt.args.namespace,
				tt.args.appSpecifier,
				tt.args.cloneOpts,
			)
			tt.assertFn(t, b, ret)
		})
	}
}

func TestRunRepoBootstrap(t *testing.T) {
	exitCalled := false
	tests := map[string]struct {
		opts     *RepoBootstrapOptions
		preFn    func(r *gitmocks.Repository, repofs fs.FS, f *kubemocks.Factory)
		assertFn func(t *testing.T, r *gitmocks.Repository, repofs fs.FS, f *kubemocks.Factory, ret error)
	}{
		"DryRun": {
			opts: &RepoBootstrapOptions{
				DryRun:           true,
				InstallationMode: installationModeFlat,
				KubeContext:      "foo",
				Namespace:        "bar",
				CloneOptions: &git.CloneOptions{
					Revision: "main",
					URL:      "https://github.com/foo/bar",
					RepoRoot: "/installation1",
					Auth:     git.Auth{Password: "test"},
				},
			},
			assertFn: func(t *testing.T, r *gitmocks.Repository, repofs fs.FS, f *kubemocks.Factory, ret error) {
				assert.NoError(t, ret)
				assert.True(t, exitCalled)
			},
		},
		"Flat installation": {
			opts: &RepoBootstrapOptions{
				InstallationMode: installationModeFlat,
				KubeContext:      "foo",
				Namespace:        "bar",
				CloneOptions: &git.CloneOptions{
					Revision: "main",
					URL:      "https://github.com/foo/bar",
					RepoRoot: "/installation1",
					Auth:     git.Auth{Password: "test"},
				},
			},
			preFn: func(r *gitmocks.Repository, repofs fs.FS, f *kubemocks.Factory) {
				mockCS := fake.NewSimpleClientset(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "argocd-initial-admin-secret",
						Namespace: "bar",
					},
					Data: map[string][]byte{
						"password": []byte("foo"),
					},
				})
				f.On("Apply", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				f.On("Wait", mock.Anything, mock.Anything).Return(nil)
				f.On("KubernetesClientSetOrDie").Return(mockCS)

				r.On("Persist", mock.Anything, mock.Anything).Return(nil)

			},
			assertFn: func(t *testing.T, r *gitmocks.Repository, repofs fs.FS, f *kubemocks.Factory, ret error) {
				assert.NoError(t, ret)
				assert.False(t, exitCalled)
				r.AssertCalled(t, "Persist", mock.Anything, mock.Anything)
				f.AssertCalled(t, "Apply", mock.Anything, "bar", mock.Anything)
				f.AssertCalled(t, "Wait", mock.Anything, mock.Anything)
				f.AssertCalled(t, "KubernetesClientSetOrDie")
				f.AssertNumberOfCalls(t, "Apply", 2)

				// bootstrap dir
				assert.True(t, repofs.ExistsOrDie(repofs.Join(
					store.Default.BootsrtrapDir,
					store.Default.ArgoCDName+".yaml",
				)))
				assert.True(t, repofs.ExistsOrDie(repofs.Join(
					store.Default.BootsrtrapDir,
					store.Default.RootAppName+".yaml",
				)))
				assert.True(t, repofs.ExistsOrDie(repofs.Join(
					store.Default.BootsrtrapDir,
					store.Default.ArgoCDName,
					"install.yaml",
				)))

				// projects
				assert.True(t, repofs.ExistsOrDie(repofs.Join(
					store.Default.ProjectsDir,
					"README.md",
				)))

				// kustomize
				assert.True(t, repofs.ExistsOrDie(repofs.Join(
					store.Default.KustomizeDir,
					"README.md",
				)))
			},
		},
		"Normal installation": {
			opts: &RepoBootstrapOptions{
				InstallationMode: installationModeNormal,
				KubeContext:      "foo",
				Namespace:        "bar",
				CloneOptions: &git.CloneOptions{
					Revision: "main",
					URL:      "https://github.com/foo/bar",
					RepoRoot: "/installation1",
					Auth:     git.Auth{Password: "test"},
				},
			},
			preFn: func(r *gitmocks.Repository, repofs fs.FS, f *kubemocks.Factory) {
				mockCS := fake.NewSimpleClientset(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "argocd-initial-admin-secret",
						Namespace: "bar",
					},
					Data: map[string][]byte{
						"password": []byte("foo"),
					},
				})
				f.On("Apply", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				f.On("Wait", mock.Anything, mock.Anything).Return(nil)
				f.On("KubernetesClientSetOrDie").Return(mockCS)

				r.On("Persist", mock.Anything, mock.Anything).Return(nil)

			},
			assertFn: func(t *testing.T, r *gitmocks.Repository, repofs fs.FS, f *kubemocks.Factory, ret error) {
				assert.NoError(t, ret)
				assert.False(t, exitCalled)
				r.AssertCalled(t, "Persist", mock.Anything, mock.Anything)
				f.AssertCalled(t, "Apply", mock.Anything, "bar", mock.Anything)
				f.AssertCalled(t, "Wait", mock.Anything, mock.Anything)
				f.AssertCalled(t, "KubernetesClientSetOrDie")
				f.AssertNumberOfCalls(t, "Apply", 2)

				// bootstrap dir
				assert.True(t, repofs.ExistsOrDie(repofs.Join(
					store.Default.BootsrtrapDir,
					store.Default.ArgoCDName+".yaml",
				)))
				assert.True(t, repofs.ExistsOrDie(repofs.Join(
					store.Default.BootsrtrapDir,
					store.Default.RootAppName+".yaml",
				)))
				assert.True(t, repofs.ExistsOrDie(repofs.Join(
					store.Default.BootsrtrapDir,
					store.Default.ArgoCDName,
					"kustomization.yaml",
				)))

				// projects
				assert.True(t, repofs.ExistsOrDie(repofs.Join(
					store.Default.ProjectsDir,
					"README.md",
				)))

				// kustomize
				assert.True(t, repofs.ExistsOrDie(repofs.Join(
					store.Default.KustomizeDir,
					"README.md",
				)))
			},
		},
	}

	orgExit := exit
	orgClone := clone
	orgRunKustomizeBuild := runKustomizeBuild
	orgArgoLogin := argocdLogin

	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			exitCalled = false
			mockRepo := &gitmocks.Repository{}
			mockFactory := &kubemocks.Factory{}
			repofs := fs.Create(memfs.New())

			if tt.preFn != nil {
				tt.preFn(mockRepo, repofs, mockFactory)
			}

			tt.opts.KubeFactory = mockFactory

			exit = func(code int) { exitCalled = true }
			clone = func(ctx context.Context, cloneOpts *git.CloneOptions, filesystem fs.FS) (git.Repository, fs.FS, error) {
				return mockRepo, repofs, nil
			}
			runKustomizeBuild = func(k *kusttypes.Kustomization) ([]byte, error) { return []byte("test"), nil }
			argocdLogin = func(opts *argocd.LoginOptions) error { return nil }

			defer func() {
				exit = orgExit
				clone = orgClone
				runKustomizeBuild = orgRunKustomizeBuild
				argocdLogin = orgArgoLogin
			}()

			tt.assertFn(t, mockRepo, repofs, mockFactory, RunRepoBootstrap(context.Background(), tt.opts))
		})
	}
}
