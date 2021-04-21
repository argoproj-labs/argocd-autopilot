package commands

import (
	"context"
	"fmt"
	"io"
	"os"

	"strings"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/fs"
	fsmocks "github.com/argoproj/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	gitmocks "github.com/argoproj/argocd-autopilot/pkg/git/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRunProjectCreate(t *testing.T) {
	tests := map[string]struct {
		opts                     *ProjectCreateOptions
		clone                    func(ctx context.Context, r *git.CloneOptions, filesystem fs.FS) (git.Repository, fs.FS, error)
		getInstallationNamespace func(fs.FS) (string, error)
		mockRepo                 git.Repository
		mockNamespace            string
		wantErr                  string
	}{
		"should handle failure in clone": {
			opts: &ProjectCreateOptions{
				Name:         "project",
				CloneOptions: &git.CloneOptions{},
			},
			clone: func(_ context.Context, _ *git.CloneOptions, _ fs.FS) (git.Repository, fs.FS, error) {
				return nil, nil, fmt.Errorf("failure clone")
			},
			wantErr: "failure clone",
		},
		"should handle failure while getting namespace": {
			opts: &ProjectCreateOptions{
				Name:         "project",
				CloneOptions: &git.CloneOptions{},
			},
			clone: func(_ context.Context, _ *git.CloneOptions, _ fs.FS) (git.Repository, fs.FS, error) {
				mockedFS := &fsmocks.FS{}
				mockedFS.On("Root").Return("/")
				return nil, mockedFS, nil
			},
			getInstallationNamespace: func(_ fs.FS) (string, error) {
				return "", fmt.Errorf("failure namespace")
			},
			wantErr: "Bootstrap folder not found, please execute ` repo bootstrap --installation-path /` command",
		},
		"should handle failure when project exists": {
			opts: &ProjectCreateOptions{
				Name:         "project",
				CloneOptions: &git.CloneOptions{},
			},
			clone: func(_ context.Context, _ *git.CloneOptions, _ fs.FS) (git.Repository, fs.FS, error) {
				mockedFS := &fsmocks.FS{}
				mockedFS.On("Root").Return("/")
				mockedFS.On("Join", "projects", "project.yaml").Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mockedFS.On("ExistsOrDie", "projects/project.yaml").Return(true)
				return nil, mockedFS, nil
			},
			getInstallationNamespace: func(_ fs.FS) (string, error) {
				return "namespace", nil
			},
			wantErr: "project 'project' already exists",
		},
		"should handle failure when writing project file": {
			opts: &ProjectCreateOptions{
				Name:         "project",
				CloneOptions: &git.CloneOptions{},
			},
			clone: func(_ context.Context, _ *git.CloneOptions, _ fs.FS) (git.Repository, fs.FS, error) {
				mockedFS := &fsmocks.FS{}
				mockedFS.On("Root").Return("/")
				mockedFS.On("Join", "projects", "project.yaml").Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mockedFS.On("ExistsOrDie", "projects/project.yaml").Return(false)
				mockedFS.On("WriteFile", "projects/project.yaml", mock.AnythingOfType("[]uint8")).Return(0, os.ErrPermission)
				return nil, mockedFS, nil
			},
			getInstallationNamespace: func(_ fs.FS) (string, error) {
				return "namespace", nil
			},
			wantErr: "failed to create project file: permission denied",
		},
		"should handle failure to persist repo": {
			opts: &ProjectCreateOptions{
				Name:         "project",
				CloneOptions: &git.CloneOptions{},
			},
			clone: func(_ context.Context, _ *git.CloneOptions, _ fs.FS) (git.Repository, fs.FS, error) {
				mockedFS := &fsmocks.FS{}
				mockedFS.On("Root").Return("/")
				mockedFS.On("Join", "projects", "project.yaml").Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mockedFS.On("ExistsOrDie", "projects/project.yaml").Return(false)
				mockedFS.On("WriteFile", "projects/project.yaml", mock.AnythingOfType("[]uint8")).Return(1, nil)
				mockedRepo := &gitmocks.Repository{}
				mockedRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{CommitMsg: "Added project project"}).Return(fmt.Errorf("failed to persist"))
				return mockedRepo, mockedFS, nil
			},
			getInstallationNamespace: func(_ fs.FS) (string, error) {
				return "namespace", nil
			},
			wantErr: "failed to persist",
		},
		"should persist repo when done": {
			opts: &ProjectCreateOptions{
				Name:         "project",
				CloneOptions: &git.CloneOptions{},
			},
			clone: func(_ context.Context, _ *git.CloneOptions, _ fs.FS) (git.Repository, fs.FS, error) {
				mockedFS := &fsmocks.FS{}
				mockedFS.On("Root").Return("/")
				mockedFS.On("Join", "projects", "project.yaml").Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mockedFS.On("ExistsOrDie", "projects/project.yaml").Return(false)
				mockedFS.On("WriteFile", "projects/project.yaml", mock.AnythingOfType("[]uint8")).Return(1, nil)
				mockedRepo := &gitmocks.Repository{}
				mockedRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{CommitMsg: "Added project project"}).Return(nil)
				return mockedRepo, mockedFS, nil
			},
			getInstallationNamespace: func(_ fs.FS) (string, error) {
				return "namespace", nil
			},
		},
	}
	origClone := clone
	origGetInstallationNamespace := getInstallationNamespace
	for ttName, tt := range tests {
		t.Run(ttName, func(t *testing.T) {
			clone = tt.clone
			getInstallationNamespace = tt.getInstallationNamespace

			err := RunProjectCreate(context.Background(), tt.opts)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
		})
	}

	clone = origClone
	getInstallationNamespace = origGetInstallationNamespace
}

func Test_generateProject(t *testing.T) {
	tests := map[string]struct {
		o                      *GenerateProjectOptions
		wantName               string
		wantNamespace          string
		wantProjectDescription string
		wantRepoURL            string
		wantRevision           string
		wantDefaultDestServer  string
		wantProject            string
		wantPath               string
	}{
		"should generate project and appset with correct values": {
			o: &GenerateProjectOptions{
				Name:              "name",
				Namespace:         "namespace",
				DefaultDestServer: "defaultDestServer",
				RepoURL:           "repoUrl",
				Revision:          "revision",
				InstallationPath:  "some/path",
			},
			wantName:               "name",
			wantNamespace:          "namespace",
			wantProjectDescription: "name project",
			wantRepoURL:            "repoUrl",
			wantRevision:           "revision",
			wantDefaultDestServer:  "defaultDestServer",
			wantPath:               "some/path/kustomize/{{appName}}/overlays/name",
		},
	}
	for ttname, tt := range tests {
		t.Run(ttname, func(t *testing.T) {
			assert := assert.New(t)
			gotProject, gotAppSet := generateProject(tt.o)
			assert.Equal(tt.wantName, gotProject.Name, "Project Name")
			assert.Equal(tt.wantNamespace, gotProject.Namespace, "Project Namespace")
			assert.Equal(tt.wantProjectDescription, gotProject.Spec.Description, "Project Description")

			assert.Equal(tt.wantName, gotAppSet.Name, "Application Set Name")
			assert.Equal(tt.wantNamespace, gotAppSet.Namespace, "Application Set Namespace")
			assert.Equal(tt.wantRepoURL, gotAppSet.Spec.Generators[0].Git.RepoURL, "Application Set Repo URL")
			assert.Equal(tt.wantRevision, gotAppSet.Spec.Generators[0].Git.Revision, "Application Set Revision")
			assert.Equal(tt.o.DefaultDestServer, gotAppSet.Spec.Generators[0].Git.Template.Spec.Destination.Server, "Application Set Default Destination Server")

			assert.Equal(tt.wantNamespace, gotAppSet.Spec.Template.Namespace, "Application Set Template Repo URL")
			assert.Equal(tt.wantName, gotAppSet.Spec.Template.Spec.Project, "Application Set Template Project")
			assert.Equal(tt.wantRepoURL, gotAppSet.Spec.Template.Spec.Source.RepoURL, "Application Set Template Repo URL")
			assert.Equal(tt.wantRevision, gotAppSet.Spec.Template.Spec.Source.TargetRevision, "Application Set Template Target Revision")
			assert.Equal(tt.wantPath, gotAppSet.Spec.Template.Spec.Source.Path, "Application Set Template Target Revision")
		})
	}
}

func Test_getInstallationNamespace(t *testing.T) {
	tests := map[string]struct {
		nsYAML   string
		beforeFn func(m *fsmocks.FS, mockedFile *fsmocks.File)
		want     string
		wantErr  string
	}{
		"should return the namespace from namespace.yaml": {
			beforeFn: func(mockedFS *fsmocks.FS, mockedFile *fsmocks.File) {
				nsYAML := `
apiVersion: v1
kind: Namespace
metadata:
  name: namespace
`
				mockedFS.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string"), "namespace.yaml").Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mockedFS.On("Open", mock.Anything).Return(mockedFile, nil)
				mockedFile.On("Read", mock.Anything).Run(func(args mock.Arguments) {
					bytes := args[0].([]byte)
					copy(bytes[:], nsYAML)
				}).Return(len(nsYAML), nil).Once()
				mockedFile.On("Read", mock.Anything).Return(0, io.EOF).Once()
			},
			want: "namespace",
		},
		"should handle file not found": {
			beforeFn: func(mockedFS *fsmocks.FS, _ *fsmocks.File) {
				mockedFS.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string"), "namespace.yaml").Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mockedFS.On("Open", mock.Anything).Return(nil, os.ErrNotExist)
			},
			wantErr: "file does not exist",
		},
		"should handle error during read": {
			beforeFn: func(mockedFS *fsmocks.FS, mockedFile *fsmocks.File) {
				mockedFS.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string"), "namespace.yaml").Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mockedFS.On("Open", mock.Anything).Return(mockedFile, nil)
				mockedFile.On("Read", mock.Anything).Return(0, fmt.Errorf("some error"))
			},
			wantErr: "failed to read namespace file: some error",
		},
		"should handle curropted namespace.yaml file": {
			beforeFn: func(mockedFS *fsmocks.FS, mockedFile *fsmocks.File) {
				nsYAML := "some string"
				mockedFS.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string"), "namespace.yaml").Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mockedFS.On("Open", mock.Anything).Return(mockedFile, nil)
				mockedFile.On("Read", mock.Anything).Run(func(args mock.Arguments) {
					bytes := args[0].([]byte)
					copy(bytes[:], nsYAML)
				}).Return(len(nsYAML), nil).Once()
				mockedFile.On("Read", mock.Anything).Return(0, io.EOF).Once()
			},
			wantErr: "failed to unmarshal namespace: error unmarshaling JSON: json: cannot unmarshal string into Go value of type v1.Namespace",
		},
	}
	for ttName, tt := range tests {
		t.Run(ttName, func(t *testing.T) {
			mockedFile := &fsmocks.File{}
			mockedFS := &fsmocks.FS{}
			tt.beforeFn(mockedFS, mockedFile)
			got, err := getInstallationNamespace(mockedFS)
			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("getInstallationNamespace() error = %v", err)
				}

				return
			}

			if got != tt.want {
				t.Errorf("getInstallationNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}
