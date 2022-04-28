package commands

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/argoproj-labs/argocd-autopilot/pkg/argocd"
	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git"
	gitmocks "github.com/argoproj-labs/argocd-autopilot/pkg/git/mocks"
	"github.com/argoproj-labs/argocd-autopilot/pkg/kube"
	kubemocks "github.com/argoproj-labs/argocd-autopilot/pkg/kube/mocks"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"

	argocdcommon "github.com/argoproj/argo-cd/v2/common"
	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/golang/mock/gomock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	kusttypes "sigs.k8s.io/kustomize/api/types"

	"github.com/go-git/go-billy/v5/memfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/assert"
)

func Test_setBootstrapOptsDefaults(t *testing.T) {
	tests := map[string]struct {
		opts     *RepoBootstrapOptions
		preFn    func()
		assertFn func(t *testing.T, opts *RepoBootstrapOptions, ret error)
	}{
		"Bad installation mode": {
			opts: &RepoBootstrapOptions{
				CloneOptions:     &git.CloneOptions{},
				InstallationMode: "foo",
			},
			assertFn: func(t *testing.T, _ *RepoBootstrapOptions, ret error) {
				assert.EqualError(t, ret, "unknown installation mode: foo")
			},
		},
		"Basic": {
			opts: &RepoBootstrapOptions{
				CloneOptions: &git.CloneOptions{},
			},
			assertFn: func(t *testing.T, opts *RepoBootstrapOptions, ret error) {
				assert.NoError(t, ret)
				assert.Equal(t, "argocd", opts.Namespace)
				assert.Equal(t, false, opts.Insecure)
				assert.Equal(t, "manifests/base", opts.AppSpecifier)
			},
		},
		"With App specifier": {
			opts: &RepoBootstrapOptions{
				CloneOptions: &git.CloneOptions{},
				AppSpecifier: "https://github.com/foo/bar",
			},
			assertFn: func(t *testing.T, opts *RepoBootstrapOptions, ret error) {
				assert.NoError(t, ret)
				assert.Equal(t, "argocd", opts.Namespace)
				assert.Equal(t, false, opts.Insecure)
				assert.Equal(t, installationModeNormal, opts.InstallationMode)
				assert.Equal(t, "https://github.com/foo/bar", opts.AppSpecifier)
			},
		},
		"Insecure": {
			opts: &RepoBootstrapOptions{
				CloneOptions:     &git.CloneOptions{},
				InstallationMode: installationModeFlat,
				Insecure:         true,
				Namespace:        "bar",
			},
			assertFn: func(t *testing.T, opts *RepoBootstrapOptions, ret error) {
				assert.NoError(t, ret)
				assert.Equal(t, "bar", opts.Namespace)
				assert.Equal(t, true, opts.Insecure)
				assert.Equal(t, installationModeFlat, opts.InstallationMode)
				assert.Equal(t, "manifests/insecure", opts.AppSpecifier)
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
		wantErr string
		preFn   func(t *testing.T, repofs fs.FS)
	}{
		"Bootstrap exists": {
			wantErr: fmt.Sprintf("folder %[1]s already exist in: /%[1]s", store.Default.BootsrtrapDir),
			preFn: func(_ *testing.T, repofs fs.FS) {
				_ = repofs.MkdirAll("bootstrap", 0666)
			},
		},
		"Projects exists": {
			wantErr: fmt.Sprintf("folder %[1]s already exist in: /%[1]s", store.Default.ProjectsDir),
			preFn: func(_ *testing.T, repofs fs.FS) {
				_ = repofs.MkdirAll("projects", 0666)
			},
		},
		"Valid": {},
	}

	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			repofs := fs.Create(memfs.New())
			if tt.preFn != nil {
				tt.preFn(t, repofs)
			}

			if err := validateRepo(repofs); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("prepare() error = %v", err)
				}

				return
			}
		})
	}
}

func Test_buildBootstrapManifests(t *testing.T) {
	type args struct {
		namespace           string
		appSpecifier        string
		cloneOpts           *git.CloneOptions
		argoCDLabels        map[string]string
		bootstrapAppsLabels map[string]string
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
					Repo: "https://github.com/foo/bar/installation1?ref=main",
					Auth: git.Auth{Password: "test"},
				},
				argoCDLabels: map[string]string{
					"name": "value",
				},
				bootstrapAppsLabels: map[string]string{
					"name": "value2",
				},
			},
			assertFn: func(t *testing.T, b *bootstrapManifests, ret error) {
				assert.NoError(t, ret)
				assert.Equal(t, []byte("test"), b.applyManifests)

				argocdApp := &argocdv1alpha1.Application{}
				assert.NoError(t, yaml.Unmarshal(b.argocdApp, argocdApp))
				assert.Equal(t, "https://github.com/foo/bar.git", argocdApp.Spec.Source.RepoURL)
				assert.Equal(t, filepath.Join("installation1", store.Default.BootsrtrapDir, store.Default.ArgoCDName), argocdApp.Spec.Source.Path)
				assert.Equal(t, "main", argocdApp.Spec.Source.TargetRevision)
				assert.Equal(t, 0, len(argocdApp.ObjectMeta.Finalizers))
				assert.Equal(t, "foo", argocdApp.Spec.Destination.Namespace)
				assert.Equal(t, store.Default.DestServer, argocdApp.Spec.Destination.Server)
				assert.Equal(t, "value", argocdApp.ObjectMeta.Labels["name"])

				bootstrapApp := &argocdv1alpha1.Application{}
				assert.NoError(t, yaml.Unmarshal(b.bootstrapApp, bootstrapApp))
				assert.Equal(t, "https://github.com/foo/bar.git", bootstrapApp.Spec.Source.RepoURL)
				assert.Equal(t, filepath.Join("installation1", store.Default.BootsrtrapDir), bootstrapApp.Spec.Source.Path)
				assert.Equal(t, "main", bootstrapApp.Spec.Source.TargetRevision)
				assert.NotEqual(t, 0, len(bootstrapApp.ObjectMeta.Finalizers))
				assert.Equal(t, "foo", bootstrapApp.Spec.Destination.Namespace)
				assert.Equal(t, store.Default.DestServer, bootstrapApp.Spec.Destination.Server)
				assert.Equal(t, "value2", bootstrapApp.ObjectMeta.Labels["name"])

				rootApp := &argocdv1alpha1.Application{}
				assert.NoError(t, yaml.Unmarshal(b.rootApp, rootApp))
				assert.Equal(t, "https://github.com/foo/bar.git", rootApp.Spec.Source.RepoURL)
				assert.Equal(t, filepath.Join("installation1", store.Default.ProjectsDir), rootApp.Spec.Source.Path)
				assert.Equal(t, "main", rootApp.Spec.Source.TargetRevision)
				assert.NotEqual(t, 0, len(rootApp.ObjectMeta.Finalizers))
				assert.Equal(t, "foo", rootApp.Spec.Destination.Namespace)
				assert.Equal(t, store.Default.DestServer, rootApp.Spec.Destination.Server)
				assert.Equal(t, "value2", rootApp.ObjectMeta.Labels["name"])

				ns := &v1.Namespace{}
				assert.NoError(t, yaml.Unmarshal(b.namespace, ns))
				assert.Equal(t, "foo", ns.ObjectMeta.Name)

				creds := &v1.Secret{}
				assert.NoError(t, yaml.Unmarshal(b.repoCreds, &creds))
				assert.Equal(t, store.Default.RepoCredsSecretName, creds.ObjectMeta.Name)
				assert.Equal(t, "foo", creds.ObjectMeta.Namespace)
				assert.Equal(t, []byte("test"), creds.Data["git_token"])
				assert.Equal(t, []byte(store.Default.GitHubUsername), creds.Data["git_username"])
			},
		},
	}

	orgRunKustomizeBuild := runKustomizeBuild
	defer func() { runKustomizeBuild = orgRunKustomizeBuild }()

	runKustomizeBuild = func(k *kusttypes.Kustomization) ([]byte, error) {
		return []byte("test"), nil
	}

	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			tt.args.cloneOpts.Parse()

			b, ret := buildBootstrapManifests(
				tt.args.namespace,
				tt.args.appSpecifier,
				tt.args.cloneOpts,
				tt.args.argoCDLabels,
				tt.args.bootstrapAppsLabels,
				nil,
			)

			tt.assertFn(t, b, ret)
		})
	}
}

func TestRunRepoBootstrap(t *testing.T) {
	exitCalled := false
	tests := map[string]struct {
		opts     *RepoBootstrapOptions
		beforeFn func(*gitmocks.MockRepository, *kubemocks.MockFactory)
		assertFn func(*testing.T, fs.FS, error)
	}{
		"DryRun": {
			opts: &RepoBootstrapOptions{
				DryRun:           true,
				InstallationMode: installationModeFlat,
				Namespace:        "bar",
				CloneOptions: &git.CloneOptions{
					Repo: "https://github.com/foo/bar/installation1?ref=main",
					Auth: git.Auth{Password: "test"},
				},
			},
			beforeFn: func(*gitmocks.MockRepository, *kubemocks.MockFactory) {},
			assertFn: func(t *testing.T, _ fs.FS, ret error) {
				assert.NoError(t, ret)
				assert.True(t, exitCalled)
			},
		},
		"Flat installation": {
			opts: &RepoBootstrapOptions{
				InstallationMode: installationModeFlat,
				Namespace:        "bar",
				CloneOptions: &git.CloneOptions{
					Repo: "https://github.com/foo/bar/installation1?ref=main",
					Auth: git.Auth{Password: "test"},
				},
			},
			beforeFn: func(r *gitmocks.MockRepository, f *kubemocks.MockFactory) {
				mockCS := fake.NewSimpleClientset(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "argocd-initial-admin-secret",
						Namespace: "bar",
					},
					Data: map[string][]byte{
						"password": []byte("foo"),
					},
				})
				r.EXPECT().Persist(gomock.Any(), gomock.Any()).Return("revision", nil)
				f.EXPECT().Apply(gomock.Any(), gomock.Any()).
					Times(2).
					Return(nil)
				f.EXPECT().Wait(gomock.Any(), gomock.Any()).Return(nil)
				f.EXPECT().KubernetesClientSetOrDie().Return(mockCS)
			},
			assertFn: func(t *testing.T, repofs fs.FS, ret error) {
				assert.NoError(t, ret)
				assert.False(t, exitCalled)

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
					store.Default.AppsDir,
					"README.md",
				)))
			},
		},
		"Normal installation": {
			opts: &RepoBootstrapOptions{
				InstallationMode: installationModeNormal,
				Namespace:        "bar",
				CloneOptions: &git.CloneOptions{
					Repo: "https://github.com/foo/bar/installation1?ref=main",
					Auth: git.Auth{Password: "test"},
				},
			},
			beforeFn: func(r *gitmocks.MockRepository, f *kubemocks.MockFactory) {
				mockCS := fake.NewSimpleClientset(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "argocd-initial-admin-secret",
						Namespace: "bar",
					},
					Data: map[string][]byte{
						"password": []byte("foo"),
					},
				})
				r.EXPECT().Persist(gomock.Any(), &git.PushOptions{CommitMsg: "Autopilot Bootstrap"}).Return("revision", nil)
				f.EXPECT().Apply(gomock.Any(), gomock.Any()).
					Times(2).
					Return(nil)
				f.EXPECT().Wait(gomock.Any(), gomock.Any()).Return(nil)
				f.EXPECT().KubernetesClientSetOrDie().Return(mockCS)
			},
			assertFn: func(t *testing.T, repofs fs.FS, ret error) {
				assert.NoError(t, ret)
				assert.False(t, exitCalled)

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
					store.Default.AppsDir,
					"README.md",
				)))
			},
		},
	}

	origExit, origGetRepo, origRunKustomizeBuild, origArgoLogin := exit, getRepo, runKustomizeBuild, argocdLogin
	defer func() {
		exit = origExit
		getRepo = origGetRepo
		runKustomizeBuild = origRunKustomizeBuild
		argocdLogin = origArgoLogin
	}()
	exit = func(_ int) { exitCalled = true }
	runKustomizeBuild = func(k *kusttypes.Kustomization) ([]byte, error) { return []byte("test"), nil }
	argocdLogin = func(opts *argocd.LoginOptions) error { return nil }

	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			r := gitmocks.NewMockRepository(ctrl)
			f := kubemocks.NewMockFactory(ctrl)
			repofs := fs.Create(memfs.New())
			exitCalled = false

			tt.beforeFn(r, f)
			tt.opts.KubeFactory = f
			getRepo = func(_ context.Context, _ *git.CloneOptions) (git.Repository, fs.FS, error) {
				return r, repofs, nil
			}

			err := RunRepoBootstrap(context.Background(), tt.opts)
			tt.assertFn(t, repofs, err)
		})
	}
}

func Test_setUninstallOptsDefaults(t *testing.T) {
	tests := map[string]struct {
		opts               RepoUninstallOptions
		want               *RepoUninstallOptions
		currentKubeContext func() (string, error)
	}{
		"Should not change anything, if all options are set": {
			opts: RepoUninstallOptions{
				Namespace:       "namespace",
				KubeContextName: "test",
			},
			want: &RepoUninstallOptions{
				Namespace:       "namespace",
				KubeContextName: "test",
			},
		},
		"Should set default argocd namespace, if it is not set": {
			opts: RepoUninstallOptions{},
			want: &RepoUninstallOptions{
				Namespace:       store.Default.ArgoCDNamespace,
				KubeContextName: "test",
			},
		},
	}
	origCurrentKubeContext := currentKubeContext
	defer func() { currentKubeContext = origCurrentKubeContext }()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			currentKubeContext = func() (string, error) {
				return "test", nil
			}

			got, _ := setUninstallOptsDefaults(tt.opts)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_deleteGitOpsFiles(t *testing.T) {
	tests := map[string]struct {
		wantErr  string
		beforeFn func() fs.FS
		assertFn func(*testing.T, fs.FS, error)
	}{
		"Should remove apps|project folders, and keep only bootstrap/DUMMY file": {
			beforeFn: func() fs.FS {
				repofs := memfs.New()
				_ = billyUtils.WriteFile(repofs, repofs.Join(store.Default.AppsDir, "some_file"), []byte{}, 0666)
				_ = billyUtils.WriteFile(repofs, repofs.Join(store.Default.BootsrtrapDir, "some_file"), []byte{}, 0666)
				_ = billyUtils.WriteFile(repofs, repofs.Join(store.Default.ProjectsDir, "some_file"), []byte{}, 0666)
				return fs.Create(repofs)
			},
			assertFn: func(t *testing.T, repofs fs.FS, err error) {
				assert.Nil(t, err)
				assert.False(t, repofs.ExistsOrDie(store.Default.AppsDir))
				assert.True(t, repofs.ExistsOrDie(repofs.Join(store.Default.BootsrtrapDir, store.Default.DummyName)))
				assert.False(t, repofs.ExistsOrDie(store.Default.ProjectsDir))
				fi, _ := repofs.ReadDir(store.Default.BootsrtrapDir)
				assert.Len(t, fi, 1)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fs := tt.beforeFn()
			err := deleteGitOpsFiles(fs)
			tt.assertFn(t, fs, err)
		})
	}
}

func Test_deleteClusterResources(t *testing.T) {
	tests := map[string]struct {
		beforeFn func(*kubemocks.MockFactory)
		assertFn func(*testing.T, kube.Factory, error)
	}{
		"Should delete all resources": {
			beforeFn: func(mf *kubemocks.MockFactory) {
				labelSelectors := []string{
					store.Default.LabelKeyAppManagedBy + "=" + store.Default.LabelValueManagedBy,
					argocdcommon.LabelKeyAppInstance + "=" + store.Default.ArgoCDName,
					store.Default.LabelKeyAppPartOf + "=" + store.Default.ArgoCDNamespace,
					store.Default.LabelKeyAppPartOf + "=" + store.Default.ArgoCDApplicationSet,
				}

				for _, labelSelector := range labelSelectors {
					mf.EXPECT().Delete(gomock.Any(), &kube.DeleteOptions{
						LabelSelector: labelSelector,
						ResourceTypes: []string{"applications", "secrets"},
					}).Return(nil)
					mf.EXPECT().Delete(gomock.Any(), &kube.DeleteOptions{
						LabelSelector: labelSelector,
						ResourceTypes: []string{
							"all",
							"configmaps",
							"secrets",
							"serviceaccounts",
							"networkpolicies",
							"rolebindings",
							"roles",
						},
					}).Return(nil)
				}
			},
			assertFn: func(t *testing.T, _ kube.Factory, err error) {
				assert.Nil(t, err)
			},
		},
		"Should fail if failed to delete argocd-autopilot resources": {
			beforeFn: func(mf *kubemocks.MockFactory) {
				labelSelectors := []string{
					store.Default.LabelKeyAppManagedBy + "=" + store.Default.LabelValueManagedBy,
				}

				for _, labelSelector := range labelSelectors {
					mf.EXPECT().Delete(gomock.Any(), &kube.DeleteOptions{
						LabelSelector: labelSelector,
						ResourceTypes: []string{"applications", "secrets"},
					}).Return(errors.New("some error"))
				}
			},
			assertFn: func(t *testing.T, _ kube.Factory, err error) {
				assert.EqualError(t, err, "failed deleting argocd-autopilot resources: some error")
			},
		},
		"Should fail if failed to delete Argo-CD resources": {
			beforeFn: func(mf *kubemocks.MockFactory) {
				mf.EXPECT().Delete(gomock.Any(), &kube.DeleteOptions{
					LabelSelector: store.Default.LabelKeyAppManagedBy + "=" + store.Default.LabelValueManagedBy,
					ResourceTypes: []string{"applications", "secrets"},
				}).Return(nil)
				mf.EXPECT().Delete(gomock.Any(), &kube.DeleteOptions{
					LabelSelector: store.Default.LabelKeyAppManagedBy + "=" + store.Default.LabelValueManagedBy,
					ResourceTypes: []string{
						"all",
						"configmaps",
						"secrets",
						"serviceaccounts",
						"networkpolicies",
						"rolebindings",
						"roles",
					},
				}).Return(errors.New("some error"))
			},
			assertFn: func(t *testing.T, _ kube.Factory, err error) {
				assert.EqualError(t, err, "failed deleting Argo-CD resources: some error")
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := kubemocks.NewMockFactory(gomock.NewController(t))
			tt.beforeFn(f)
			err := deleteClusterResources(context.Background(), &deleteClusterResourcesOptions{
				KubeFactory: f,
				Timeout:     0,
				FastExit:    true,
			})
			tt.assertFn(t, f, err)
		})
	}
}

func TestRunRepoUninstall(t *testing.T) {
	tests := map[string]struct {
		currentKubeContextErr error
		getRepoErr            error
		wantErr               string
		beforeFn              func(*gitmocks.MockRepository, *kubemocks.MockFactory)
	}{
		"Should fail if getCurrentKubeContext fails": {
			currentKubeContextErr: errors.New("some error"),
			wantErr:               "some error",
		},
		"Should fail if getRepo fails": {
			getRepoErr: errors.New("some error"),
			wantErr:    "some error",
		},
		"Should fail if Persist fails": {
			wantErr: "some error",
			beforeFn: func(r *gitmocks.MockRepository, _ *kubemocks.MockFactory) {
				r.EXPECT().Persist(gomock.Any(), &git.PushOptions{CommitMsg: "Autopilot Uninstall"}).
					Return("", errors.New("some error"))
			},
		},
		"Should fail if Wait fails": {
			wantErr: "some error",
			beforeFn: func(r *gitmocks.MockRepository, f *kubemocks.MockFactory) {
				r.EXPECT().Persist(gomock.Any(), &git.PushOptions{CommitMsg: "Autopilot Uninstall"}).
					Return("revision", nil)
				f.EXPECT().Wait(gomock.Any(), gomock.Any()).Return(errors.New("some error"))
			},
		},
		"Should fail if deleteClusterResources fails": {
			wantErr: "failed deleting argocd-autopilot resources: some error",
			beforeFn: func(r *gitmocks.MockRepository, f *kubemocks.MockFactory) {
				r.EXPECT().Persist(gomock.Any(), &git.PushOptions{CommitMsg: "Autopilot Uninstall"}).
					Return("revision", nil)
				f.EXPECT().Wait(gomock.Any(), gomock.Any()).
					Return(nil)
				f.EXPECT().Delete(gomock.Any(), &kube.DeleteOptions{
					LabelSelector:   store.Default.LabelKeyAppManagedBy + "=" + store.Default.LabelValueManagedBy,
					ResourceTypes:   []string{"applications", "secrets"},
					WaitForDeletion: false,
				}).
					Return(errors.New("some error"))
				f.EXPECT().Delete(gomock.Any(), &kube.DeleteOptions{
					LabelSelector: store.Default.LabelKeyAppManagedBy + "=" + store.Default.LabelValueManagedBy,
					ResourceTypes: []string{
						"all",
						"configmaps",
						"secrets",
						"serviceaccounts",
						"networkpolicies",
						"rolebindings",
						"roles",
					}, WaitForDeletion: false,
				}).
					Times(0)
			},
		},
		"Should fail if 2nd Persist fails": {
			wantErr: "some error",
			beforeFn: func(r *gitmocks.MockRepository, f *kubemocks.MockFactory) {
				r.EXPECT().Persist(gomock.Any(), &git.PushOptions{CommitMsg: "Autopilot Uninstall"}).
					Return("revision", nil)
				r.EXPECT().Persist(gomock.Any(), &git.PushOptions{CommitMsg: "Autopilot Uninstall, deleted leftovers"}).
					Return("", errors.New("some error"))
				f.EXPECT().Wait(gomock.Any(), gomock.Any()).
					Return(nil)

				labelSelectors := []string{
					store.Default.LabelKeyAppManagedBy + "=" + store.Default.LabelValueManagedBy,
					argocdcommon.LabelKeyAppInstance + "=" + store.Default.ArgoCDName,
					store.Default.LabelKeyAppPartOf + "=" + store.Default.ArgoCDNamespace,
					store.Default.LabelKeyAppPartOf + "=" + store.Default.ArgoCDApplicationSet,
				}

				for _, labelSelector := range labelSelectors {
					f.EXPECT().Delete(gomock.Any(), &kube.DeleteOptions{
						LabelSelector: labelSelector,
						ResourceTypes: []string{"applications", "secrets"},
					}).
						Return(nil)
					f.EXPECT().Delete(gomock.Any(), &kube.DeleteOptions{
						LabelSelector: labelSelector,
						ResourceTypes: []string{
							"all",
							"configmaps",
							"secrets",
							"serviceaccounts",
							"networkpolicies",
							"rolebindings",
							"roles",
						},
					}).
						Return(nil)
				}
			},
		},
		"Should succeed if no errors": {
			beforeFn: func(r *gitmocks.MockRepository, f *kubemocks.MockFactory) {
				r.EXPECT().Persist(gomock.Any(), &git.PushOptions{CommitMsg: "Autopilot Uninstall"}).
					Return("revision", nil)
				r.EXPECT().Persist(gomock.Any(), &git.PushOptions{CommitMsg: "Autopilot Uninstall, deleted leftovers"}).
					Return("", nil)
				f.EXPECT().Wait(gomock.Any(), gomock.Any()).
					Return(nil)

				labelSelectors := []string{
					store.Default.LabelKeyAppManagedBy + "=" + store.Default.LabelValueManagedBy,
					argocdcommon.LabelKeyAppInstance + "=" + store.Default.ArgoCDName,
					store.Default.LabelKeyAppPartOf + "=" + store.Default.ArgoCDNamespace,
					store.Default.LabelKeyAppPartOf + "=" + store.Default.ArgoCDApplicationSet,
				}

				for _, labelSelector := range labelSelectors {
					f.EXPECT().Delete(gomock.Any(), &kube.DeleteOptions{
						LabelSelector: labelSelector,
						ResourceTypes: []string{"applications", "secrets"},
					}).
						Return(nil)

					f.EXPECT().Delete(gomock.Any(), &kube.DeleteOptions{
						LabelSelector: labelSelector,
						ResourceTypes: []string{
							"all",
							"configmaps",
							"secrets",
							"serviceaccounts",
							"networkpolicies",
							"rolebindings",
							"roles",
						},
					}).
						Return(nil)
				}
			},
		},
	}

	origGetRepo, origCurrentKubeContext := getRepo, currentKubeContext
	defer func() { getRepo, currentKubeContext = origGetRepo, origCurrentKubeContext }()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			r := gitmocks.NewMockRepository(ctrl)
			repofs := fs.Create(memfs.New())
			f := kubemocks.NewMockFactory(ctrl)

			if tt.beforeFn != nil {
				tt.beforeFn(r, f)
			}

			getRepo = func(_ context.Context, _ *git.CloneOptions) (git.Repository, fs.FS, error) {
				if tt.getRepoErr != nil {
					return nil, nil, tt.getRepoErr
				}

				return r, repofs, nil
			}
			currentKubeContext = func() (string, error) {
				if tt.currentKubeContextErr != nil {
					return "", tt.currentKubeContextErr
				}

				return "context", nil
			}

			opts := &RepoUninstallOptions{
				CloneOptions: &git.CloneOptions{
					Repo: "https://github.com/owner/name",
				},
				KubeFactory: f,
				FastExit:    true,
			}
			opts.CloneOptions.Parse()
			err := RunRepoUninstall(context.Background(), opts)
			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("RunRepoUninstall() error = %v", err)
				}

				return
			}
		})
	}
}
