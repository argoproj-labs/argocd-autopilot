package commands

import (
	"bytes"
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
	"github.com/go-git/go-billy/v5"
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
		writeFileError           error
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
				return nil, mockedFS, nil
			},
			getInstallationNamespace: func(_ fs.FS) (string, error) {
				return "namespace", nil
			},
			writeFileError: os.ErrPermission,
			wantErr:        "failed to create project file: permission denied",
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
	origWriteFile := writeFile
	for ttName, tt := range tests {
		t.Run(ttName, func(t *testing.T) {
			clone = tt.clone
			getInstallationNamespace = tt.getInstallationNamespace
			writeFile = func(fs billy.Basic, filename string, data []byte) error {
				return tt.writeFileError
			}
			if err := RunProjectCreate(context.Background(), tt.opts); tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}

	clone = origClone
	getInstallationNamespace = origGetInstallationNamespace
	writeFile = origWriteFile
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
		beforeFn func(fs.FS) (fs.FS, error)
	}{
		"should return error if project file doesn't exist": {
			name:    "prod.yaml",
			wantErr: "prod.yaml not found",
		},
		"should failed when 2 files not found": {
			name:    "prod.yaml",
			wantErr: "expected 2 files when splitting prod.yaml",
			beforeFn: func(f fs.FS) (fs.FS, error) {
				err := writeFile(f, "prod.yaml", []byte("content"))
				if err != nil {
					return nil, err
				}
				return f, nil
			},
		},
		"should return AppProject": {
			name: "prod.yaml",
			beforeFn: func(f fs.FS) (fs.FS, error) {
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
				err := writeFile(f, "prod.yaml", joinedYAML)
				if err != nil {
					return nil, err
				}
				return f, nil
			},
			want: &argocdv1alpha1.AppProject{
				ObjectMeta: v1.ObjectMeta{
					Name:      "prod",
					Namespace: "ns",
				},
			},
		},
	}
	for tName, tt := range tests {
		t.Run(tName, func(t *testing.T) {
			repofs := fs.Create(memfs.New())
			if tt.beforeFn != nil {
				_, err := tt.beforeFn(repofs)
				assert.NoError(t, err)
			}
			got, _, err := getProjectInfoFromFile(repofs, tt.name)
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

func TestRunProjectList(t *testing.T) {
	type args struct {
		ctx  context.Context
		opts *ProjectListOptions
	}
	tests := map[string]struct {
		args                   args
		wantErr                bool
		prepareRepo            func(ctx context.Context, o *BaseOptions) (git.Repository, fs.FS, error)
		glob                   func(fs billy.Filesystem, pattern string) ([]string, error)
		getProjectInfoFromFile func(fs fs.FS, name string) (*argocdv1alpha1.AppProject, *appsetv1alpha1.ApplicationSpec, error)
		assertFn               func(t *testing.T, str string)
	}{
		"should print to table": {
			args: args{
				opts: &ProjectListOptions{
					BaseOptions: BaseOptions{},
					Out:         &bytes.Buffer{},
				},
			},
			glob: func(_ billy.Filesystem, _ string) ([]string, error) {
				res := make([]string, 0, 1)
				res = append(res, "prod.yaml")
				return res, nil
			},
			prepareRepo: func(_ context.Context, _ *BaseOptions) (git.Repository, fs.FS, error) {
				memFS := fs.Create(memfs.New())
				return nil, memFS, nil
			},
			getProjectInfoFromFile: func(_ fs.FS, _ string) (*argocdv1alpha1.AppProject, *appsetv1alpha1.ApplicationSpec, error) {
				appProj := &argocdv1alpha1.AppProject{
					ObjectMeta: v1.ObjectMeta{
						Name:      "prod",
						Namespace: "ns",
					},
				}
				return appProj, nil, nil
			},
			assertFn: func(t *testing.T, str string) {
				assert.Contains(t, str, "NAME  NAMESPACE  CLUSTER  \n")
				assert.Contains(t, str, "prod  ns  ")

			},
		},
	}
	origPrepareRepo := prepareRepo
	origGlob := glob
	origGetProjectInfoFromFile := getProjectInfoFromFile
	for tName, tt := range tests {
		t.Run(tName, func(t *testing.T) {
			prepareRepo = tt.prepareRepo
			glob = tt.glob
			getProjectInfoFromFile = tt.getProjectInfoFromFile
			if err := RunProjectList(tt.args.ctx, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("RunProjectList() error = %v, wantErr %v", err, tt.wantErr)
			}
			b := tt.args.opts.Out.(*bytes.Buffer)
			tt.assertFn(t, b.String())
			prepareRepo = origPrepareRepo
			glob = origGlob
			getProjectInfoFromFile = origGetProjectInfoFromFile
		})
	}
}

func TestRunProjectDelete(t *testing.T) {
	tests := map[string]struct {
		projectName string
		prepareErr  string
		globMatches []string
		wantErr     string
		beforeFn    func(mfs *fsmocks.FS, mr *gitmocks.Repository)
		glob        func(fs billy.Filesystem, pattern string) (matches []string, err error)
		removeAll   func(fs billy.Basic, path string) error
	}{
		"Should fail when clone fails": {
			projectName: "project",
			prepareErr:  "some error",
			wantErr:     "some error",
		},
		"Should fail when glob fails": {
			projectName: "project",
			glob: func(_ billy.Filesystem, pattern string) (matches []string, err error) {
				assert.Equal(t, "kustomize/*/overlays/project", pattern)
				return nil, fmt.Errorf("some error")
			},
			wantErr: "failed to run glob on 'kustomize/*/overlays/project': some error",
		},
		"Should fail when ReadDir fails": {
			projectName: "project",
			wantErr:     "failed to read overlays directory 'kustomize/app1/overlays': some error",
			beforeFn: func(mfs *fsmocks.FS, _ *gitmocks.Repository) {
				mfs.On("ReadDir", "kustomize/app1/overlays").Return(nil, fmt.Errorf("some error"))
			},
			glob: func(_ billy.Filesystem, pattern string) (matches []string, err error) {
				assert.Equal(t, "kustomize/*/overlays/project", pattern)
				return []string{"kustomize/app1/overlays/project"}, nil
			},
		},
		"Should fail when removeAll fails": {
			projectName: "project",
			wantErr:     "failed to delete directory 'kustomize/app1': some error",
			beforeFn: func(mfs *fsmocks.FS, _ *gitmocks.Repository) {
				mfs.On("ReadDir", "kustomize/app1/overlays").Return([]os.FileInfo{nil}, nil)
			},
			glob: func(_ billy.Filesystem, pattern string) (matches []string, err error) {
				assert.Equal(t, "kustomize/*/overlays/project", pattern)
				return []string{"kustomize/app1/overlays/project"}, nil
			},
			removeAll: func(_ billy.Basic, path string) error {
				assert.Equal(t, "kustomize/app1", path)
				return fmt.Errorf("some error")
			},
		},
		"Should fail when Remove fails": {
			projectName: "project",
			wantErr:     "failed to delete project 'project': some error",
			beforeFn: func(mfs *fsmocks.FS, _ *gitmocks.Repository) {
				mfs.On("ReadDir", "kustomize/app1/overlays").Return([]os.FileInfo{nil}, nil)
				mfs.On("Remove", "projects/project.yaml").Return(fmt.Errorf("some error"))
			},
			glob: func(_ billy.Filesystem, pattern string) (matches []string, err error) {
				assert.Equal(t, "kustomize/*/overlays/project", pattern)
				return []string{"kustomize/app1/overlays/project"}, nil
			},
			removeAll: func(_ billy.Basic, path string) error {
				assert.Equal(t, "kustomize/app1", path)
				return nil
			},
		},
		"Should fail when persist fails": {
			projectName: "project",
			wantErr:     "failed to push to repo: some error",
			beforeFn: func(mfs *fsmocks.FS, mr *gitmocks.Repository) {
				mfs.On("ReadDir", "kustomize/app1/overlays").Return([]os.FileInfo{nil}, nil)
				mfs.On("Remove", "projects/project.yaml").Return(nil)
				mr.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(fmt.Errorf("some error"))
			},
			glob: func(_ billy.Filesystem, pattern string) (matches []string, err error) {
				assert.Equal(t, "kustomize/*/overlays/project", pattern)
				return []string{"kustomize/app1/overlays/project"}, nil
			},
			removeAll: func(_ billy.Basic, path string) error {
				assert.Equal(t, "kustomize/app1", path)
				return nil
			},
		},
		"Should remove entire app folder, if it contains only one overlay": {
			projectName: "project",
			beforeFn: func(mfs *fsmocks.FS, mr *gitmocks.Repository) {
				mfs.On("ReadDir", "kustomize/app1/overlays").Return([]os.FileInfo{nil}, nil)
				mfs.On("Remove", "projects/project.yaml").Return(nil)
				mr.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(nil)
			},
			glob: func(_ billy.Filesystem, pattern string) (matches []string, err error) {
				assert.Equal(t, "kustomize/*/overlays/project", pattern)
				return []string{"kustomize/app1/overlays/project"}, nil
			},
			removeAll: func(_ billy.Basic, path string) error {
				assert.Equal(t, "kustomize/app1", path)
				return nil
			},
		},
		"Should remove only overlay, if app contains more overlays": {
			projectName: "project",
			beforeFn: func(mfs *fsmocks.FS, mr *gitmocks.Repository) {
				mfs.On("ReadDir", "kustomize/app1/overlays").Return([]os.FileInfo{nil, nil}, nil)
				mfs.On("Remove", "projects/project.yaml").Return(nil)
				mr.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(nil)
			},
			glob: func(_ billy.Filesystem, pattern string) (matches []string, err error) {
				assert.Equal(t, "kustomize/*/overlays/project", pattern)
				return []string{"kustomize/app1/overlays/project"}, nil
			},
			removeAll: func(_ billy.Basic, path string) error {
				assert.Equal(t, "kustomize/app1/overlays/project", path)
				return nil
			},
		},
		"Should handle multiple apps": {
			projectName: "project",
			beforeFn: func(mfs *fsmocks.FS, mr *gitmocks.Repository) {
				mfs.On("ReadDir", "kustomize/app1/overlays").Return([]os.FileInfo{nil, nil}, nil)
				mfs.On("ReadDir", "kustomize/app2/overlays").Return([]os.FileInfo{nil}, nil)
				mfs.On("Remove", "projects/project.yaml").Return(nil)
				mr.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(nil)
			},
			glob: func(_ billy.Filesystem, pattern string) (matches []string, err error) {
				assert.Equal(t, "kustomize/*/overlays/project", pattern)
				return []string{"kustomize/app1/overlays/project", "kustomize/app2/overlays/project"}, nil
			},
			removeAll: func(_ billy.Basic, path string) error {
				assert.Contains(t, []string{
					"kustomize/app1/overlays/project",
					"kustomize/app2",
				}, path)
				return nil
			},
		},
	}
	origGlob := glob
	origPrepareRepo := prepareRepo
	origRemoveAll := removeAll
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockFS := &fsmocks.FS{}
			mockRepo := &gitmocks.Repository{}
			mockFS.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(func(elem ...string) string {
				return strings.Join(elem, "/")
			})
			mockFS.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(func(elem ...string) string {
				return strings.Join(elem, "/")
			})
			prepareRepo = func(_ context.Context, _ *BaseOptions) (git.Repository, fs.FS, error) {
				if tt.prepareErr != "" {
					return nil, nil, fmt.Errorf(tt.prepareErr)
				}

				return mockRepo, mockFS, nil
			}
			glob = tt.glob
			removeAll = tt.removeAll
			if tt.beforeFn != nil {
				tt.beforeFn(mockFS, mockRepo)
			}

			opts := &BaseOptions{
				ProjectName: tt.projectName,
			}
			if err := RunProjectDelete(context.Background(), opts); err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("prepare() error = %v", err)
				}
			}
		})
	}

	glob = origGlob
	prepareRepo = origPrepareRepo
	removeAll = origRemoveAll
}
