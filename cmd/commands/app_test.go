package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argocd-autopilot/pkg/application"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	fsmocks "github.com/argoproj/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	gitmocks "github.com/argoproj/argocd-autopilot/pkg/git/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/go-git/go-billy/v5/memfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_getCommitMsg(t *testing.T) {
	tests := map[string]struct {
		appName     string
		projectName string
		root        string
		expected    string
	}{
		"On root": {
			appName:     "foo",
			projectName: "bar",
			root:        "",
			expected:    "installed app 'foo' on project 'bar'",
		},
		"On installation path": {
			appName:     "foo",
			projectName: "bar",
			root:        "foo/bar",
			expected:    "installed app 'foo' on project 'bar' installation-path: 'foo/bar'",
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			m := &fsmocks.FS{}
			m.On("Root").Return(tt.root)
			opts := &AppCreateOptions{
				BaseOptions: BaseOptions{
					ProjectName: tt.projectName,
				},
				AppOpts: &application.CreateOptions{
					AppName: tt.appName,
				},
			}
			got := getCommitMsg(opts, m)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func Test_getConfigFileFromPath(t *testing.T) {
	tests := map[string]struct {
		appName  string
		want     *application.Config
		wantErr  string
		beforeFn func(repofs fs.FS, appName string) fs.FS
		assertFn func(t *testing.T, conf *application.Config)
	}{
		"should return config.json": {
			want: &application.Config{
				AppName: "test",
			},
			appName: "test",
			beforeFn: func(repofs fs.FS, appName string) fs.FS {
				conf := application.Config{AppName: appName}
				b, _ := json.Marshal(&conf)
				_ = billyUtils.WriteFile(repofs, fmt.Sprintf("%s/config.json", appName), b, 0666)
				return repofs
			},
			assertFn: func(t *testing.T, conf *application.Config) {
				assert.Equal(t, conf.AppName, "test")
			},
		},
		"should fail if config.json is missing": {
			appName: "test",
			wantErr: "failed to read file 'test/config.json'",
		},
		"should fail if config.json failed to unmarshal": {
			appName: "test",
			wantErr: "failed to unmarshal file 'test/config.json'",
			beforeFn: func(repofs fs.FS, appName string) fs.FS {
				_ = billyUtils.WriteFile(repofs, fmt.Sprintf("%s/config.json", appName), []byte{}, 0666)
				return repofs
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			repofs := fs.Create(memfs.New())
			if tt.beforeFn != nil {
				repofs = tt.beforeFn(repofs, tt.appName)
			}

			got, err := getConfigFileFromPath(repofs, tt.appName)
			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("getConfigFileFromPath() error = %v", err)
				}

				return
			}

			if tt.assertFn != nil {
				tt.assertFn(t, got)
			}
		})
	}
}

func TestRunAppDelete(t *testing.T) {
	tests := map[string]struct {
		appName     string
		projectName string
		global      bool
		wantErr     string
		prepareRepo func() (git.Repository, fs.FS, error)
		assertFn    func(t *testing.T, repo git.Repository, repofs fs.FS)
	}{
		"Should fail when clone fails": {
			wantErr: "some error",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				return nil, nil, fmt.Errorf("some error")
			},
		},
		"Should fail when app does not exist": {
			appName: "app",
			wantErr: "application 'app' not found",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				return nil, fs.Create(memfs.New()), nil
			},
		},
		"Should fail if deletion of entire app directory fails": {
			appName: "app",
			global:  true,
			wantErr: fmt.Sprintf("failed to delete directory '%s': some error", filepath.Join(store.Default.AppsDir, "app")),
			prepareRepo: func() (git.Repository, fs.FS, error) {
				mfs := &fsmocks.FS{}
				mfs.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				path := filepath.Join(store.Default.AppsDir, "app")
				mfs.On("ExistsOrDie", path).Return(true)
				mfs.On("Remove", path).Return(fmt.Errorf("some error"))
				mfs.On("Stat", path).Return(nil, fmt.Errorf("some error"))
				return nil, mfs, nil
			},
		},
		"Should remove entire app directory when global flag is set": {
			appName: "app",
			global:  true,
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted app 'app'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")))
			},
		},
		"Should fail when overlay does not exist": {
			appName:     "app",
			projectName: "project",
			wantErr:     "application 'app' not found in project 'project'",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project2"), 0666)
				return nil, fs.Create(memfs), nil
			},
		},
		"Should fail if ReadDir fails": {
			appName:     "app",
			projectName: "project",
			wantErr:     fmt.Sprintf("failed to read overlays directory '%s': some error", filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir)),
			prepareRepo: func() (git.Repository, fs.FS, error) {
				mfs := &fsmocks.FS{}
				mfs.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mfs.On("ExistsOrDie", filepath.Join(store.Default.AppsDir, "app")).Return(true)
				mfs.On("ExistsOrDie", filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir)).Return(true)
				mfs.On("ExistsOrDie", filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project")).Return(true)
				mfs.On("ReadDir", filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir)).Return(nil, fmt.Errorf("some error"))
				return nil, mfs, nil
			},
		},
		"Should delete only overlay directory of a kustApp, if there are more overlays": {
			appName:     "app",
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project2"), 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted app 'app' from project 'project'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.True(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir)))
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project")))
			},
		},
		"Should delete entire app directory of a kustApp, if there are no more overlays": {
			appName:     "app",
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted app 'app'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")))
			},
		},
		"Should delete only project directory of a dirApp, if there are more projects": {
			appName:     "app",
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", "project"), 0666)
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", "project2"), 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted app 'app' from project 'project'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.True(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")))
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app", "project")))
			},
		},
		"Should delete entire app directory of a dirApp": {
			appName:     "app",
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", "project"), 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted app 'app'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")))
			},
		},
		"Should fail if Persist fails": {
			appName: "app",
			global:  true,
			wantErr: "failed to push to repo: some error",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted app 'app'",
				}).Return(fmt.Errorf("some error"))
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")))
			},
		},
	}
	origPrepareRepo := prepareRepo
	defer func() { prepareRepo = origPrepareRepo }()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				repo   git.Repository
				repofs fs.FS
			)

			prepareRepo = func(_ context.Context, _ *BaseOptions) (git.Repository, fs.FS, error) {
				var err error
				repo, repofs, err = tt.prepareRepo()
				return repo, repofs, err
			}
			opts := &AppDeleteOptions{
				BaseOptions: BaseOptions{
					ProjectName: tt.projectName,
					FS:          fs.Create(memfs.New()),
				},
				AppName: tt.appName,
				Global:  tt.global,
			}
			if err := RunAppDelete(context.Background(), opts); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("RunAppDelete() error = %v", err)
				}

				return
			}

			if tt.assertFn != nil {
				tt.assertFn(t, repo, repofs)
			}
		})
	}
}

func Test_getProjectDestServer(t *testing.T) {
	tests := map[string]struct {
		want     string
		wantErr  string
		beforeFn func() fs.FS
	}{
		"Should return dest server from file": {
			want: "https://dest.server",
			beforeFn: func() fs.FS {
				repofs := fs.Create(memfs.New())
				project := &argocdv1alpha1.AppProject{
					TypeMeta: metav1.TypeMeta{
						Kind:       argocdv1alpha1.AppProjectSchemaGroupVersionKind.Kind,
						APIVersion: argocdv1alpha1.AppProjectSchemaGroupVersionKind.GroupVersion().String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "project",
						Annotations: map[string]string{
							store.Default.DestServerAnnotation: "https://dest.server",
						},
					},
				}
				_ = repofs.WriteYamls(repofs.Join(store.Default.ProjectsDir, "project.yaml"), project)
				return repofs
			},
		},
		"Should fail if project file is not available": {
			wantErr: "failed to unmarshal project: file does not exist",
			beforeFn: func() fs.FS {
				repofs := fs.Create(memfs.New())
				return repofs
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repofs := tt.beforeFn()
			got, err := getProjectDestServer(repofs, "project")
			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("getProjectDestServer() error = %v", err)
				}

				return
			}

			if got != tt.want {
				t.Errorf("getProjectDestServer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_setAppOptsDefaults(t *testing.T) {
	tests := map[string]struct {
		opts     *AppCreateOptions
		wantErr  string
		beforeFn func() fs.FS
		assertFn func(*testing.T, *AppCreateOptions)
	}{
		"Should change nothing if all fields are set": {
			opts: &AppCreateOptions{
				AppOpts: &application.CreateOptions{
					DestServer:    "https://dest.server",
					DestNamespace: "namespace",
					AppType:       application.AppTypeKustomize,
				},
			},
			assertFn: func(t *testing.T, opts *AppCreateOptions) {
				assert.Equal(t, "https://dest.server", opts.AppOpts.DestServer)
				assert.Equal(t, "namespace", opts.AppOpts.DestNamespace)
				assert.Equal(t, application.AppTypeKustomize, opts.AppOpts.AppType)
			},
		},
		"Should set namespace to 'default', if empty": {
			opts: &AppCreateOptions{
				AppOpts: &application.CreateOptions{
					DestServer: "https://dest.server",
					AppType:    application.AppTypeKustomize,
				},
			},
			assertFn: func(t *testing.T, opts *AppCreateOptions) {
				assert.Equal(t, "default", opts.AppOpts.DestNamespace)
			},
		},
		"Should read server from project, if empty": {
			opts: &AppCreateOptions{
				BaseOptions: BaseOptions{
					ProjectName: "project",
				},
				AppOpts: &application.CreateOptions{
					DestNamespace: "namespace",
					AppType:       application.AppTypeKustomize,
				},
			},
			beforeFn: func() fs.FS {
				repofs := fs.Create(memfs.New())
				project := &argocdv1alpha1.AppProject{
					TypeMeta: metav1.TypeMeta{
						Kind:       argocdv1alpha1.AppProjectSchemaGroupVersionKind.Kind,
						APIVersion: argocdv1alpha1.AppProjectSchemaGroupVersionKind.GroupVersion().String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "project",
						Annotations: map[string]string{
							store.Default.DestServerAnnotation: "https://dest.server",
						},
					},
				}
				_ = repofs.WriteYamls(repofs.Join(store.Default.ProjectsDir, "project.yaml"), project)
				return repofs
			},
			assertFn: func(t *testing.T, opts *AppCreateOptions) {
				assert.Equal(t, "https://dest.server", opts.AppOpts.DestServer)
			},
		},
		"Should read server from project, if set to default": {
			opts: &AppCreateOptions{
				BaseOptions: BaseOptions{
					ProjectName: "project",
				},
				AppOpts: &application.CreateOptions{
					DestServer:    store.Default.DestServer,
					DestNamespace: "namespace",
					AppType:       application.AppTypeKustomize,
				},
			},
			beforeFn: func() fs.FS {
				repofs := fs.Create(memfs.New())
				project := &argocdv1alpha1.AppProject{
					TypeMeta: metav1.TypeMeta{
						Kind:       argocdv1alpha1.AppProjectSchemaGroupVersionKind.Kind,
						APIVersion: argocdv1alpha1.AppProjectSchemaGroupVersionKind.GroupVersion().String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "project",
						Annotations: map[string]string{
							store.Default.DestServerAnnotation: "https://dest.server",
						},
					},
				}
				_ = repofs.WriteYamls(repofs.Join(store.Default.ProjectsDir, "project.yaml"), project)
				return repofs
			},
			assertFn: func(t *testing.T, opts *AppCreateOptions) {
				assert.Equal(t, "https://dest.server", opts.AppOpts.DestServer)
			},
		},
		"Should infer appType from repo, if empty": {
			opts: &AppCreateOptions{
				BaseOptions: BaseOptions{
					CloneOptions: &git.CloneOptions{
						Auth: git.Auth{},
					},
				},
				AppOpts: &application.CreateOptions{
					AppSpecifier:  "github.com/owner/repo/some/path",
					DestServer:    "https://dest.server",
					DestNamespace: "namespace",
				},
			},
			beforeFn: func() fs.FS {
				clone = func(_ context.Context, _ *git.CloneOptions, _ fs.FS) (git.Repository, fs.FS, error) {
					return nil, fs.Create(memfs.New()), nil
				}

				return nil
			},
			assertFn: func(t *testing.T, opts *AppCreateOptions) {
				assert.Equal(t, application.AppTypeDirectory, opts.AppOpts.AppType)
			},
		},
		"Should fail if can't read server from project": {
			opts: &AppCreateOptions{
				BaseOptions: BaseOptions{
					ProjectName: "project",
				},
				AppOpts: &application.CreateOptions{
					DestNamespace: "namespace",
					AppType:       application.AppTypeKustomize,
				},
			},
			wantErr: "failed to unmarshal project: file does not exist",
			beforeFn: func() fs.FS {
				return fs.Create(memfs.New())
			},
		},
		"Should fail if can't infer appType": {
			opts: &AppCreateOptions{
				BaseOptions: BaseOptions{
					CloneOptions: &git.CloneOptions{
						Auth: git.Auth{},
					},
				},
				AppOpts: &application.CreateOptions{
					AppSpecifier:  "github.com/owner/repo/some/path",
					DestServer:    "https://dest.server",
					DestNamespace: "namespace",
				},
			},
			wantErr: "some error",
			beforeFn: func() fs.FS {
				clone = func(_ context.Context, _ *git.CloneOptions, _ fs.FS) (git.Repository, fs.FS, error) {
					return nil, nil, fmt.Errorf("some error")
				}

				return nil
			},
		},
	}
	origClone := clone
	defer func() { clone = origClone }()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var repofs fs.FS
			if tt.beforeFn != nil {
				repofs = tt.beforeFn()
			}

			if err := setAppOptsDefaults(context.Background(), repofs, tt.opts); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("setAppOptsDefaults() error = %v", err)
				}

				return
			}

			if tt.assertFn != nil {
				tt.assertFn(t, tt.opts)
			}
		})
	}
}
