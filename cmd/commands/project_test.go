package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"testing"

	appsetv1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	fsmocks "github.com/argoproj/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	gitmocks "github.com/argoproj/argocd-autopilot/pkg/git/mocks"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/ghodss/yaml"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			wantErr: util.Doc("Bootstrap folder not found, please execute `<BIN> repo bootstrap --installation-path /` command"),
		},
		"should handle failure when project exists": {
			opts: &ProjectCreateOptions{
				Name:         "project",
				CloneOptions: &git.CloneOptions{},
			},
			clone: func(_ context.Context, _ *git.CloneOptions, _ fs.FS) (git.Repository, fs.FS, error) {
				mockedFS := &fsmocks.FS{}
				mockedFS.On("Root").Return("/")
				mockedFS.On("Join", "projects", "project.yaml").Return("projects/project.yaml")
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
				mockedFS.On("Join", "projects", "project.yaml").Return("projects/project.yaml")
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
				mockedFS.On("Join", "projects", "project.yaml").Return("projects/project.yaml")
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
				mockedFS.On("Join", "projects", "project.yaml").Return("projects/project.yaml")
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
			assert.Equal(tt.o.DefaultDestServer, gotProject.Annotations[store.Default.DestServerAnnotation], "Application Set Default Destination Server")

			assert.Equal(tt.wantName, gotAppSet.Name, "Application Set Name")
			assert.Equal(tt.wantNamespace, gotAppSet.Namespace, "Application Set Namespace")
			assert.Equal(tt.wantRepoURL, gotAppSet.Spec.Generators[0].Git.RepoURL, "Application Set Repo URL")
			assert.Equal(tt.wantRevision, gotAppSet.Spec.Generators[0].Git.Revision, "Application Set Revision")

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
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: argo-cd
spec:
  destination:
    namespace: namespace
    server: https://kubernetes.default.svc
  source:
    path: manifests
    repoURL: https://github.com/owner/name
`
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
				mockedFS.On("Open", mock.Anything).Return(nil, os.ErrNotExist)
			},
			wantErr: "file does not exist",
		},
		"should handle error during read": {
			beforeFn: func(mockedFS *fsmocks.FS, mockedFile *fsmocks.File) {
				mockedFS.On("Open", mock.Anything).Return(mockedFile, nil)
				mockedFile.On("Read", mock.Anything).Return(0, fmt.Errorf("some error"))
			},
			wantErr: "failed to read namespace file: some error",
		},
		"should handle corrupted namespace.yaml file": {
			beforeFn: func(mockedFS *fsmocks.FS, mockedFile *fsmocks.File) {
				nsYAML := "some string"
				mockedFS.On("Open", mock.Anything).Return(mockedFile, nil)
				mockedFile.On("Read", mock.Anything).Run(func(args mock.Arguments) {
					bytes := args[0].([]byte)
					copy(bytes[:], nsYAML)
				}).Return(len(nsYAML), nil).Once()
				mockedFile.On("Read", mock.Anything).Return(0, io.EOF).Once()
			},
			wantErr: "failed to unmarshal namespace: error unmarshaling JSON: json: cannot unmarshal string into Go value of type v1alpha1.Application",
		},
	}
	for ttName, tt := range tests {
		t.Run(ttName, func(t *testing.T) {
			mockFile := &fsmocks.File{}
			mockFS := &fsmocks.FS{}
			mockFS.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(func(elem ...string) string {
				return strings.Join(elem, "/")
			})
			tt.beforeFn(mockFS, mockFile)
			got, err := getInstallationNamespace(mockFS)
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

func Test_getProjectInfoFromFile(t *testing.T) {
	tests := map[string]struct {
		name     string
		want     *argocdv1alpha1.AppProject
		wantErr  string
		beforeFn func(fs.FS) fs.FS
	}{
		"should return error if project file doesn't exist": {
			name:    "prod.yaml",
			wantErr: "prod.yaml not found",
		},
		"should failed when 2 files not found": {
			name:    "prod.yaml",
			wantErr: "expected 2 files when splitting prod.yaml",
			beforeFn: func(f fs.FS) fs.FS {
				f.WriteFile("prod.yaml", []byte("content"))
				return f
			},
		},
		"should return AppProject": {
			name: "prod.yaml",
			beforeFn: func(f fs.FS) fs.FS {
				appProj := argocdv1alpha1.AppProject{
					ObjectMeta: v1.ObjectMeta{
						Name:      "prod",
						Namespace: "ns",
					},
				}
				appSet := appsetv1alpha1.ApplicationSpec{}
				projectYAML, _ := yaml.Marshal(&appProj)
				appsetYAML, _ := yaml.Marshal(&appSet)
				joinedYAML := util.JoinManifests(projectYAML, appsetYAML)
				f.WriteFile("prod.yaml", joinedYAML)
				return f
			},
			want: &argocdv1alpha1.AppProject{
				ObjectMeta: v1.ObjectMeta{
					Name:      "prod",
					Namespace: "ns",
				},
			},
		},
	}
	for tNAME, tt := range tests {
		t.Run(tNAME, func(t *testing.T) {
			repofs := fs.Create(memfs.New())
			if tt.beforeFn != nil {
				repofs = tt.beforeFn(repofs)
			}
			got, err := getProjectInfoFromFile(repofs, tt.name)
			if (err != nil) && tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getProjectInfoFromFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
