package commands

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/fs"
	fsmocks "github.com/argoproj/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	gitmocks "github.com/argoproj/argocd-autopilot/pkg/git/mocks"

	"github.com/stretchr/testify/assert"
)

func TestBaseOptions_preRun(t *testing.T) {
	tests := map[string]struct {
		projectName string
		cloneErr    string
		wantErr     string
		beforeFn    func(m *fsmocks.FS)
	}{
		"Should complete when no errors are returned": {
			projectName: "",
			cloneErr:    "",
			wantErr:     "",
			beforeFn: func(m *fsmocks.FS) {
				m.On("Root").Return("/")
				m.On("ExistsOrDie", "bootstrap").Return(true)
			},
		},
		"Should fail when clone fails": {
			projectName: "project",
			cloneErr:    "some error",
			wantErr:     "some error",
			beforeFn:    func(m *fsmocks.FS) {},
		},
		"Should fail when there is no bootstrap at repo root": {
			projectName: "",
			cloneErr:    "",
			wantErr:     "Bootstrap directory not found, please execute `repo bootstrap` command",
			beforeFn: func(m *fsmocks.FS) {
				m.On("Root").Return("/")
				m.On("ExistsOrDie", "bootstrap").Return(false)
			},
		},
		"Should fail when there is no bootstrap at instllation path": {
			projectName: "",
			cloneErr:    "",
			wantErr:     "Bootstrap directory not found, please execute `repo bootstrap --installation-path /some/path` command",
			beforeFn: func(m *fsmocks.FS) {
				m.On("Root").Return("/some/path")
				m.On("ExistsOrDie", "bootstrap").Return(false)
			},
		},
		"Should not validate project existence, if no projectName is supplied": {
			projectName: "",
			cloneErr:    "",
			wantErr:     "",
			beforeFn: func(m *fsmocks.FS) {
				m.On("Root").Return("/")
				m.On("ExistsOrDie", "bootstrap").Return(true)
			},
		},
		"Should fail when project does not exist": {
			projectName: "project",
			cloneErr:    "",
			wantErr:     "project 'project' not found, please execute `argocd-autopilot project create project`",
			beforeFn: func(m *fsmocks.FS) {
				m.On("Root").Return("/")
				m.On("ExistsOrDie", "bootstrap").Return(true)
				m.On("Join", "projects", "project.yaml").Return(func(elem ...string) string {
					return "projects/project.yaml"
				})
				m.On("ExistsOrDie", "projects/project.yaml").Return(false)
			},
		},
	}
	origClone := clone
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := &gitmocks.Repository{}
			mockFS := &fsmocks.FS{}
			tt.beforeFn(mockFS)
			clone = func(_ context.Context, _ *git.CloneOptions, _ fs.FS) (git.Repository, fs.FS, error) {
				var err error
				if tt.cloneErr != "" {
					err = fmt.Errorf(tt.cloneErr)
				}

				return mockRepo, mockFS, err
			}
			o := &BaseOptions{
				CloneOptions: &git.CloneOptions{},
				FS:           nil,
				ProjectName:  tt.projectName,
			}
			gotRepo, gotFS, err := o.preRun(context.Background())
			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("prepare() error = %v", err)
				}

				return
			}

			if !reflect.DeepEqual(gotRepo, mockRepo) {
				t.Errorf("BaseOptions.clone() got = %v, want %v", gotRepo, mockRepo)
			}

			if !reflect.DeepEqual(gotFS, mockFS) {
				t.Errorf("BaseOptions.clone() got1 = %v, want %v", gotFS, mockFS)
			}
		})
	}

	clone = origClone
}
