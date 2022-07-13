package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git"
	gitmocks "github.com/argoproj-labs/argocd-autopilot/pkg/git/mocks"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"

	"github.com/go-git/go-billy/v5/memfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func Test_prepareRepo(t *testing.T) {
	tests := map[string]struct {
		projectName string
		wantErr     string
		getRepo     func(*testing.T) (git.Repository, fs.FS, error)
		assertFn    func(*testing.T, git.Repository, fs.FS)
	}{
		"Should complete when no errors are returned": {
			getRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				repofs := fs.Create(memfs.New())
				_ = repofs.MkdirAll(store.Default.BootsrtrapDir, 0666)
				return gitmocks.NewMockRepository(gomock.NewController(t)), repofs, nil
			},
			assertFn: func(t *testing.T, r git.Repository, fs fs.FS) {
				assert.NotNil(t, r)
				assert.NotNil(t, fs)
			},
		},
		"Should fail when clone fails": {
			wantErr: "failed cloning the repository: some error",
			getRepo: func(*testing.T) (git.Repository, fs.FS, error) {
				return nil, nil, errors.New("some error")
			},
		},
		"Should fail when there is no bootstrap at repo root": {
			wantErr: "bootstrap directory not found, please execute `repo bootstrap` command",
			getRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				return gitmocks.NewMockRepository(gomock.NewController(t)), fs.Create(memfs.New()), nil
			},
			assertFn: func(t *testing.T, r git.Repository, fs fs.FS) {
				assert.NotNil(t, r)
				assert.NotNil(t, fs)
			},
		},
		"Should validate project existence if a projectName is supplied": {
			projectName: "project",
			getRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				repofs := fs.Create(memfs.New())
				_ = repofs.MkdirAll(store.Default.BootsrtrapDir, 0666)
				_ = billyUtils.WriteFile(repofs, repofs.Join(store.Default.ProjectsDir, "project.yaml"), []byte{}, 0666)
				return gitmocks.NewMockRepository(gomock.NewController(t)), repofs, nil
			},
			assertFn: func(t *testing.T, r git.Repository, fs fs.FS) {
				assert.NotNil(t, r)
				assert.NotNil(t, fs)
			},
		},
		"Should fail when project does not exist": {
			projectName: "project",
			wantErr:     "project 'project' not found, please execute `argocd-autopilot project create project`",
			getRepo: func(t *testing.T) (git.Repository, fs.FS, error) {
				repofs := fs.Create(memfs.New())
				_ = repofs.MkdirAll(store.Default.BootsrtrapDir, 0666)
				return gitmocks.NewMockRepository(gomock.NewController(t)), repofs, nil
			},
		},
	}
	origGetRepo := getRepo
	defer func() { getRepo = origGetRepo }()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			getRepo = func(_ context.Context, _ *git.CloneOptions) (git.Repository, fs.FS, error) {
				return tt.getRepo(t)
			}
			r, fs, err := prepareRepo(context.Background(), &git.CloneOptions{}, tt.projectName)
			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("prepare() error = %v", err)
				}

				return
			}

			tt.assertFn(t, r, fs)
		})
	}
}
