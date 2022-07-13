package commands

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/argoproj-labs/argocd-autopilot/pkg/application"
	appmocks "github.com/argoproj-labs/argocd-autopilot/pkg/application/mocks"
	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	fsmocks "github.com/argoproj-labs/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git"
	gitmocks "github.com/argoproj-labs/argocd-autopilot/pkg/git/mocks"
	kubemocks "github.com/argoproj-labs/argocd-autopilot/pkg/kube/mocks"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/go-git/go-billy/v5/memfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRunAppCreate(t *testing.T) {
	tests := map[string]struct {
		appsRepo                 string
		timeout                  time.Duration
		wantErr                  string
		setAppOptsDefaultsErr    error
		parseAppErr              error
		createFilesErr           error
		beforeFn                 func(f *kubemocks.MockFactory)
		prepareRepo              func(*testing.T) (git.Repository, fs.FS, error)
		getRepo                  func(*testing.T, *git.CloneOptions) (git.Repository, fs.FS, error)
		getInstallationNamespace func(repofs fs.FS) (string, error)
	}{
		"Should fail when clone fails": {
			wantErr: "some error",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				return nil, nil, fmt.Errorf("some error")
			},
		},
		"Should fail if srcClone fails": {
			appsRepo: "https://github.com/owner/other_name",
			wantErr:  "some error",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				return nil, nil, nil
			},
			getRepo: func(_ *testing.T, _ *git.CloneOptions) (git.Repository, fs.FS, error) {
				return nil, nil, fmt.Errorf("some error")
			},
		},
		"Should use cloneOpts password for srcCloneOpts, if required": {
			appsRepo: "https://github.com/owner/other_name/path?ref=branch",
			wantErr:  "some error",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				return nil, nil, nil
			},
			getRepo: func(t *testing.T, opts *git.CloneOptions) (git.Repository, fs.FS, error) {
				assert.Equal(t, "https://github.com/owner/other_name.git", opts.URL())
				assert.Equal(t, "branch", opts.Revision())
				assert.Equal(t, "path", opts.Path())
				assert.Equal(t, "password", opts.Auth.Password)
				return nil, nil, fmt.Errorf("some error")
			},
		},
		"Should fail if setAppOptsDefaults fails": {
			wantErr: "some error",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				return nil, nil, nil
			},
			setAppOptsDefaultsErr: fmt.Errorf("some error"),
		},
		"Should fail if app parse fails": {
			wantErr: "failed to parse application from flags: some error",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				return nil, nil, nil
			},
			parseAppErr: errors.New("some error"),
		},
		"Should fail if app already exist in project": {
			wantErr:        fmt.Errorf("application 'app' already exists in project 'project': %w", application.ErrAppAlreadyInstalledOnProject).Error(),
			createFilesErr: application.ErrAppAlreadyInstalledOnProject,
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				return mockRepo, fs.Create(memfs), nil
			},
		},
		"Should fail if file creation fails": {
			wantErr:        "some error",
			createFilesErr: errors.New("some error"),
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				return mockRepo, fs.Create(memfs), nil
			},
		},
		"Should fail if commiting to appsRepo fails": {
			appsRepo: "https://github.com/owner/other_name",
			wantErr:  "failed to push to apps repo: some error",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				return mockRepo, fs.Create(memfs), nil
			},
			getRepo: func(_ *testing.T, _ *git.CloneOptions) (git.Repository, fs.FS, error) {
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "installed app 'app' on project 'project' installation-path: '/'",
				}).
					Times(1).
					Return("", fmt.Errorf("some error"))
				return mockRepo, fs.Create(memfs.New()), nil
			},
		},
		"Should fail if commiting to gitops repo fails": {
			wantErr: "failed to push to gitops repo: some error",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "installed app 'app' on project 'project' installation-path: '/'",
				}).
					Times(1).
					Return("", fmt.Errorf("some error"))
				return mockRepo, fs.Create(memfs), nil
			},
		},
		"Should fail if getInstallationNamespace fails": {
			timeout: 1,
			wantErr: "failed to get application namespace: some error",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "installed app 'app' on project 'project' installation-path: '/'",
				}).
					Times(1).
					Return("revision", nil)
				return mockRepo, fs.Create(memfs), nil
			},
			getInstallationNamespace: func(repofs fs.FS) (string, error) {
				return "", errors.New("some error")
			},
		},
		"Should fail if waiting fails": {
			timeout: 1,
			wantErr: "failed waiting for application to sync: some error",
			beforeFn: func(f *kubemocks.MockFactory) {
				f.EXPECT().Wait(gomock.Any(), gomock.Any()).
					Times(1).
					Return(errors.New("some error"))
			},
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "installed app 'app' on project 'project' installation-path: '/'",
				}).
					Times(1).
					Return("revision", nil)
				return mockRepo, fs.Create(memfs), nil
			},
			getInstallationNamespace: func(repofs fs.FS) (string, error) {
				return "namespace", nil
			},
		},
		"Should Persist to both repos, if required": {
			appsRepo: "https://github.com/owner/other_name",
			wantErr:  "failed to push to gitops repo: some error",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "installed app 'app' on project 'project' installation-path: '/'",
				}).
					Times(1).
					Return("revision", nil)
				return mockRepo, fs.Create(memfs), nil
			},
			getRepo: func(t *testing.T, _ *git.CloneOptions) (git.Repository, fs.FS, error) {
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "installed app 'app' on project 'project' installation-path: '/'",
				}).
					Times(1).
					Return("revision", nil)
				return mockRepo, fs.Create(memfs.New()), nil
			},
		},
		"Should Persist to a single repo, if required": {
			wantErr: "failed to push to gitops repo: some error",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "installed app 'app' on project 'project' installation-path: '/'",
				}).
					Times(1).
					Return("revision", nil)
				return mockRepo, fs.Create(memfs), nil
			},
		},
		"Should wait succesfully and complete": {
			timeout: 1,
			beforeFn: func(f *kubemocks.MockFactory) {
				f.EXPECT().Wait(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil)
			},
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "installed app 'app' on project 'project' installation-path: '/'",
				}).
					Times(1).
					Return("revision", nil)
				return mockRepo, fs.Create(memfs), nil
			},
			getInstallationNamespace: func(repofs fs.FS) (string, error) {
				return "namespace", nil
			},
		},
	}
	origPrepareRepo, origGetRepo, origSetAppOptsDefault, origAppParse, origGetInstallationNamespace := prepareRepo, getRepo, setAppOptsDefaults, parseApp, getInstallationNamespace
	defer func() {
		prepareRepo = origPrepareRepo
		getRepo = origGetRepo
		setAppOptsDefaults = origSetAppOptsDefault
		parseApp = origAppParse
		getInstallationNamespace = origGetInstallationNamespace
	}()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var (
				gitopsRepo git.Repository
				appsRepo   git.Repository
			)

			ctrl := gomock.NewController(t)
			f := kubemocks.NewMockFactory(ctrl)
			if tt.beforeFn != nil {
				tt.beforeFn(f)
			}
			prepareRepo = func(_ context.Context, _ *git.CloneOptions, _ string) (git.Repository, fs.FS, error) {
				var (
					repofs fs.FS
					err    error
				)
				gitopsRepo, repofs, err = tt.prepareRepo(t)
				return gitopsRepo, repofs, err
			}
			getRepo = func(_ context.Context, cloneOpts *git.CloneOptions) (git.Repository, fs.FS, error) {
				var (
					repofs fs.FS
					err    error
				)
				appsRepo, repofs, err = tt.getRepo(t, cloneOpts)
				return appsRepo, repofs, err
			}
			setAppOptsDefaults = func(_ context.Context, _ fs.FS, _ *AppCreateOptions) error {
				return tt.setAppOptsDefaultsErr
			}
			parseApp = func(_ *application.CreateOptions, _, _, _, _ string) (application.Application, error) {
				if tt.parseAppErr != nil {
					return nil, tt.parseAppErr
				}

				app := appmocks.NewMockApplication(ctrl)
				app.EXPECT().Name().Return("app").AnyTimes()
				app.EXPECT().CreateFiles(gomock.Any(), gomock.Any(), "project").Return(tt.createFilesErr).AnyTimes()
				return app, nil
			}
			getInstallationNamespace = tt.getInstallationNamespace
			opts := &AppCreateOptions{
				Timeout: tt.timeout,
				CloneOpts: &git.CloneOptions{
					Repo: "https://github.com/owner/name",
					Auth: git.Auth{
						Password: "password",
					},
				},
				AppsCloneOpts: &git.CloneOptions{
					Repo: tt.appsRepo,
				},
				ProjectName: "project",
				AppOpts: &application.CreateOptions{
					AppName:      "app",
					AppType:      application.AppTypeDirectory,
					AppSpecifier: "https://github.com/owner/name/manifests",
				},
				KubeFactory: f,
			}

			opts.CloneOpts.Parse()
			opts.AppsCloneOpts.Parse()
			if err := RunAppCreate(context.Background(), opts); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("RunAppCreate() error = %v", err)
				}

				return
			}
		})
	}
}

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
			m := fsmocks.NewMockFS(gomock.NewController(t))
			m.EXPECT().Root().Return(tt.root).AnyTimes()
			opts := &AppCreateOptions{
				ProjectName: tt.projectName,
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
				_ = repofs.WriteJson(fmt.Sprintf("%s/config.json", appName), conf)
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
		prepareRepo func(*testing.T) (git.Repository, fs.FS, error)
		assertFn    func(t *testing.T, repo git.Repository, repofs fs.FS)
	}{
		"Should fail when clone fails": {
			wantErr: "some error",
			prepareRepo: func(*testing.T) (git.Repository, fs.FS, error) {
				return nil, nil, fmt.Errorf("some error")
			},
		},
		"Should fail when app does not exist": {
			appName: "app",
			wantErr: "application 'app' not found",
			prepareRepo: func(*testing.T) (git.Repository, fs.FS, error) {
				return nil, fs.Create(memfs.New()), nil
			},
		},
		"Should fail if deletion of entire app directory fails": {
			appName: "app",
			global:  true,
			wantErr: fmt.Sprintf("failed to delete directory '%s': some error", filepath.Join(store.Default.AppsDir, "app")),
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				mfs := fsmocks.NewMockFS(gomock.NewController(t))
				path := filepath.Join(store.Default.AppsDir, "app")
				mfs.EXPECT().Join(gomock.Any()).
					Times(1).
					DoAndReturn(func(elem ...string) string {
						return strings.Join(elem, "/")
					})
				mfs.EXPECT().ExistsOrDie(path).Return(true)
				mfs.EXPECT().Remove(path).Return(fmt.Errorf("some error"))
				mfs.EXPECT().Stat(path).Return(nil, fmt.Errorf("some error"))
				return nil, mfs, nil
			},
		},
		"Should remove entire app directory when global flag is set": {
			appName: "app",
			global:  true,
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "Deleted app 'app'",
				}).
					Times(1).
					Return("revision", nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, _ git.Repository, repofs fs.FS) {
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")))
			},
		},
		"Should fail when overlay does not exist": {
			appName:     "app",
			projectName: "project",
			wantErr:     "application 'app' not found in project 'project'",
			prepareRepo: func(*testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project2"), 0666)
				return nil, fs.Create(memfs), nil
			},
		},
		"Should fail if ReadDir fails": {
			appName:     "app",
			projectName: "project",
			wantErr:     fmt.Sprintf("failed to read overlays directory '%s': some error", filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir)),
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				mfs := fsmocks.NewMockFS(gomock.NewController(t))
				mfs.EXPECT().Join(gomock.Any()).
					Times(3).
					DoAndReturn(func(elem ...string) string {
						return strings.Join(elem, "/")
					})
				mfs.EXPECT().ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")).
					Times(1).
					Return(true)
				mfs.EXPECT().ExistsOrDie(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir)).
					Times(1).
					Return(true)
				mfs.EXPECT().ExistsOrDie(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project")).
					Times(1).
					Return(true)
				mfs.EXPECT().ReadDir(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir)).
					Times(1).
					Return(nil, fmt.Errorf("some error"))
				return nil, mfs, nil
			},
		},
		"Should delete only overlay directory of a kustApp, if there are more overlays": {
			appName:     "app",
			projectName: "project",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project2"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "Deleted app 'app' from project 'project'",
				}).
					Times(1).
					Return("revision", nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, _ git.Repository, repofs fs.FS) {
				assert.True(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir)))
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project")))
			},
		},
		"Should delete entire app directory of a kustApp, if there are no more overlays": {
			appName:     "app",
			projectName: "project",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "Deleted app 'app'",
				}).
					Times(1).
					Return("revision", nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, _ git.Repository, repofs fs.FS) {
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")))
			},
		},
		"Should delete only project directory of a dirApp, if there are more projects": {
			appName:     "app",
			projectName: "project",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", "project"), 0666)
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", "project2"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "Deleted app 'app' from project 'project'",
				}).
					Times(1).
					Return("revision", nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, _ git.Repository, repofs fs.FS) {
				assert.True(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")))
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app", "project")))
			},
		},
		"Should delete entire app directory of a dirApp": {
			appName:     "app",
			projectName: "project",
			prepareRepo: func(*testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "Deleted app 'app'",
				}).
					Times(1).
					Return("revision", nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, _ git.Repository, repofs fs.FS) {
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app")))
			},
		},
		"Should fail if Persist fails": {
			appName: "app",
			global:  true,
			wantErr: "failed to push to repo: some error",
			prepareRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := gitmocks.NewMockRepository(gomock.NewController(t))
				mockRepo.EXPECT().Persist(gomock.Any(), &git.PushOptions{
					CommitMsg: "Deleted app 'app'",
				}).
					Times(1).
					Return("", fmt.Errorf("some error"))
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, _ git.Repository, repofs fs.FS) {
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

			prepareRepo = func(_ context.Context, _ *git.CloneOptions, _ string) (git.Repository, fs.FS, error) {
				var err error
				repo, repofs, err = tt.prepareRepo(t)
				return repo, repofs, err
			}
			opts := &AppDeleteOptions{
				ProjectName: tt.projectName,
				AppName:     tt.appName,
				Global:      tt.global,
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
				ProjectName: "project",
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
				ProjectName: "project",
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
				CloneOpts: &git.CloneOptions{
					Auth: git.Auth{},
				},
				AppOpts: &application.CreateOptions{
					AppSpecifier:  "github.com/owner/repo/some/path",
					DestServer:    "https://dest.server",
					DestNamespace: "namespace",
				},
			},
			beforeFn: func() fs.FS {
				getRepo = func(_ context.Context, _ *git.CloneOptions) (git.Repository, fs.FS, error) {
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
				ProjectName: "project",
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
				CloneOpts: &git.CloneOptions{
					Auth: git.Auth{},
				},
				AppOpts: &application.CreateOptions{
					AppSpecifier:  "github.com/owner/repo/some/path",
					DestServer:    "https://dest.server",
					DestNamespace: "namespace",
				},
			},
			wantErr: "some error",
			beforeFn: func() fs.FS {
				getRepo = func(_ context.Context, _ *git.CloneOptions) (git.Repository, fs.FS, error) {
					return nil, nil, fmt.Errorf("some error")
				}

				return nil
			},
		},
	}
	origGetRepo := getRepo
	defer func() { getRepo = origGetRepo }()
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
