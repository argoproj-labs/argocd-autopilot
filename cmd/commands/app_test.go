package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

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
			wantErr: fmt.Sprintf("failed to delete directory '%s': some error", filepath.Join(store.Default.KustomizeDir, "app")),
			prepareRepo: func() (git.Repository, fs.FS, error) {
				mfs := &fsmocks.FS{}
				mfs.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				path := filepath.Join(store.Default.KustomizeDir, "app")
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
				_ = memfs.MkdirAll(filepath.Join(store.Default.KustomizeDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted app 'app'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.KustomizeDir, "app")))
			},
		},
		"Should fail when overlay does not exist": {
			appName:     "app",
			projectName: "project",
			wantErr:     "application 'app' not found in project 'project'",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.KustomizeDir, "app", store.Default.OverlaysDir, "project2"), 0666)
				return nil, fs.Create(memfs), nil
			},
		},
		"Should fail if ReadDir fails": {
			appName:     "app",
			projectName: "project",
			wantErr:     fmt.Sprintf("failed to read overlays directory '%s': some error", filepath.Join(store.Default.KustomizeDir, "app", store.Default.OverlaysDir)),
			prepareRepo: func() (git.Repository, fs.FS, error) {
				mfs := &fsmocks.FS{}
				mfs.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mfs.On("ExistsOrDie", filepath.Join(store.Default.KustomizeDir, "app")).Return(true)
				mfs.On("ExistsOrDie", filepath.Join(store.Default.KustomizeDir, "app", store.Default.OverlaysDir, "project")).Return(true)
				mfs.On("ReadDir", filepath.Join(store.Default.KustomizeDir, "app", store.Default.OverlaysDir)).Return(nil, fmt.Errorf("some error"))
				return nil, mfs, nil
			},
		},
		"Should delete only overlay directory, if there are more overlays": {
			appName:     "app",
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.KustomizeDir, "app", store.Default.OverlaysDir, "project"), 0666)
				_ = memfs.MkdirAll(filepath.Join(store.Default.KustomizeDir, "app", store.Default.OverlaysDir, "project2"), 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted app 'app' from project 'project'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.True(t, repofs.ExistsOrDie(filepath.Join(store.Default.KustomizeDir, "app", store.Default.OverlaysDir)))
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.KustomizeDir, "app", store.Default.OverlaysDir, "project")))
			},
		},
		"Should delete entire app directory, if there are no more overlays": {
			appName:     "app",
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.KustomizeDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted app 'app'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.KustomizeDir, "app")))
			},
		},
		"Should fail if Persist fails": {
			appName: "app",
			global:  true,
			wantErr: "failed to push to repo: some error",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.KustomizeDir, "app", store.Default.OverlaysDir, "project"), 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted app 'app'",
				}).Return(fmt.Errorf("some error"))
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.KustomizeDir, "app")))
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
					t.Errorf("prepare() error = %v", err)
				}

				return
			}

			if tt.assertFn != nil {
				tt.assertFn(t, repo, repofs)
			}
		})
	}
}
