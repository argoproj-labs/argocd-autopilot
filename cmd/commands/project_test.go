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
	memfs "github.com/go-git/go-billy/v5/memfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRunProjectCreate(t *testing.T) {
	tests := map[string]struct {
		projectName              string
		wantErr                  string
		getInstallationNamespace func(repofs fs.FS) (string, error)
		prepareRepo              func() (git.Repository, fs.FS, error)
		assertFn                 func(t *testing.T, repo git.Repository, repofs fs.FS)
	}{
		"should handle failure in prepare repo": {
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				return nil, nil, fmt.Errorf("failure clone")
			},
			wantErr: "failure clone",
		},
		"should handle failure while getting namespace": {
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				return nil, fs.Create(memfs), nil
			},
			getInstallationNamespace: func(_ fs.FS) (string, error) {
				return "", fmt.Errorf("failure namespace")
			},
			wantErr: util.Doc("Bootstrap folder not found, please execute `<BIN> repo bootstrap --installation-path /` command"),
		},
		"should handle failure when project exists": {
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "projects/project.yaml", []byte{}, 0666)
				return nil, fs.Create(memfs), nil
			},
			getInstallationNamespace: func(_ fs.FS) (string, error) {
				return "namespace", nil
			},
			wantErr: "project 'project' already exists",
		},
		"should handle failure when writing project file": {
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				mockedFS := &fsmocks.FS{}
				mockedFS.On("Root").Return("/")
				mockedFS.On("Join", "projects", "project.yaml").Return("projects/project.yaml")
				mockedFS.On("ExistsOrDie", "projects/project.yaml").Return(false)
				mockedFS.On("OpenFile", "projects/project.yaml", mock.AnythingOfType("int"), mock.AnythingOfType("FileMode")).Return(nil, os.ErrPermission)
				return nil, mockedFS, nil
			},
			getInstallationNamespace: func(_ fs.FS) (string, error) {
				return "namespace", nil
			},
			wantErr: "failed to create project file: permission denied",
		},
		"should handle failure to persist repo": {
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				mockedRepo := &gitmocks.Repository{}
				mockedRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{CommitMsg: "Added project project"}).Return(fmt.Errorf("failed to persist"))
				return mockedRepo, fs.Create(memfs), nil
			},
			getInstallationNamespace: func(_ fs.FS) (string, error) {
				return "namespace", nil
			},
			wantErr: "failed to persist",
		},
		"should persist repo when done": {
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				mockedRepo := &gitmocks.Repository{}
				mockedRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{CommitMsg: "Added project project"}).Return(nil)
				return mockedRepo, fs.Create(memfs), nil
			},
			getInstallationNamespace: func(_ fs.FS) (string, error) {
				return "namespace", nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				exists := repofs.ExistsOrDie("projects/project.yaml")
				assert.True(t, exists)
			},
		},
	}
	origPrepareRepo := prepareRepo
	origGetInstallationNamespace := getInstallationNamespace
	defer func() {
		prepareRepo = origPrepareRepo
		getInstallationNamespace = origGetInstallationNamespace
	}()
	for ttName, tt := range tests {
		t.Run(ttName, func(t *testing.T) {
			var (
				repo   git.Repository
				repofs fs.FS
			)

			opts := &ProjectCreateOptions{
				Name: tt.projectName,
				BaseOptions: BaseOptions{
					CloneOptions: &git.CloneOptions{},
				},
			}
			prepareRepo = func(_ context.Context, _ *BaseOptions) (git.Repository, fs.FS, error) {
				var err error
				repo, repofs, err = tt.prepareRepo()
				return repo, repofs, err
			}
			getInstallationNamespace = tt.getInstallationNamespace
			if err := RunProjectCreate(context.Background(), opts); tt.wantErr != "" {
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
				err := billyUtils.WriteFile(f, "prod.yaml", []byte("content"), 0666)
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
				err := billyUtils.WriteFile(f, "prod.yaml", joinedYAML, 0666)
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
	tests := map[string]struct {
		opts                   *ProjectListOptions
		wantErr                string
		prepareRepo            func(ctx context.Context, o *BaseOptions) (git.Repository, fs.FS, error)
		getProjectInfoFromFile func(fs fs.FS, name string) (*argocdv1alpha1.AppProject, *appsetv1alpha1.ApplicationSpec, error)
		assertFn               func(t *testing.T, out io.Writer)
	}{
		"should print to table": {
			opts: &ProjectListOptions{
				BaseOptions: BaseOptions{},
				Out:         &bytes.Buffer{},
			},
			prepareRepo: func(_ context.Context, _ *BaseOptions) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "projects/prod.yaml", []byte{}, 0666)
				return nil, fs.Create(memfs), nil
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
			assertFn: func(t *testing.T, out io.Writer) {
				str := out.(*bytes.Buffer).String()
				assert.Contains(t, str, "NAME  NAMESPACE  CLUSTER  \n")
				assert.Contains(t, str, "prod  ns  ")
			},
		},
	}
	origPrepareRepo := prepareRepo
	origGetProjectInfoFromFile := getProjectInfoFromFile
	defer func() {
		prepareRepo = origPrepareRepo
		getProjectInfoFromFile = origGetProjectInfoFromFile
	}()
	for tName, tt := range tests {
		t.Run(tName, func(t *testing.T) {
			prepareRepo = tt.prepareRepo
			getProjectInfoFromFile = tt.getProjectInfoFromFile

			if err := RunProjectList(context.Background(), tt.opts); tt.wantErr != "" {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("prepare() error = %v", err)
				}

				return
			}

			tt.assertFn(t, tt.opts.Out)
		})
	}
}

func TestRunProjectDelete(t *testing.T) {
	tests := map[string]struct {
		projectName string
		wantErr     string
		prepareRepo func() (git.Repository, fs.FS, error)
		assertFn    func(t *testing.T, repo git.Repository, repofs fs.FS)
	}{
		"Should fail when clone fails": {
			projectName: "project",
			wantErr:     "some error",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				return nil, nil, fmt.Errorf("some error")
			},
		},
		// "Should fail when ReadDir fails": {
		// 	projectName: "project",
		// 	wantErr:     "failed to read overlays directory 'kustomize/app1/overlays': some error",
		// 	prepareRepo: func(_ context.Context, o *BaseOptions) (git.Repository, fs.FS, error) {
		// 		_ = o.FS.MkdirAll("kustomize/app1/overlays/project", 0666)
		// 		mockfs := &fsmocks.FS{}
		// 		mockfs.On("ReadDir", "kustomize/app1/overlays").Return(nil, fmt.Errorf("some error"))
		// 		mockfs.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(o.FS.Join)
		// 		mockfs.On("Lstat", mock.AnythingOfType("string")).Return(
		// 			func(filename string) os.FileInfo {
		// 				fi, _ := o.FS.Lstat(filename)
		// 				return fi
		// 			},
		// 			func(filename string) error {
		// 				_, err := o.FS.Lstat(filename)
		// 				return err
		// 			},
		// 		)
		// 		mockfs.On("Stat", mock.AnythingOfType("string")).Return(
		// 			func(filename string) os.FileInfo {
		// 				fi, _ := o.FS.Stat(filename)
		// 				return fi
		// 			},
		// 			func(filename string) error {
		// 				_, err := o.FS.Stat(filename)
		// 				return err
		// 			},
		// 		)
		// 		mockfs.On("ReadDir", mock.AnythingOfType("string")).Return(
		// 			func(filename string) []os.FileInfo {
		// 				fi, _ := o.FS.ReadDir(filename)
		// 				return fi
		// 			},
		// 			func(filename string) error {
		// 				_, err := o.FS.ReadDir(filename)
		// 				return err
		// 			},
		// 		)
		// 		return nil, fs.Create(mockfs), nil
		// 	},
		//},
		// "Should fail when failed to delete app directory": {
		// 	projectName: "project",
		// 	wantErr:     "failed to delete directory 'kustomize/app1': some error",
		// 	prepareRepo: func(_ context.Context, o *BaseOptions) (git.Repository, fs.FS, error) {
		// 		_ = o.FS.MkdirAll("kustomize/app1/overlays/project", 0666)
		// 		mockfs := &fsmocks.FS{}
		// 		mockfs.On("Remove", "kustomize/app1").Return(fmt.Errorf("some error"))
		// 		mockfs.On("Stat", "kustomize/app1").Return(nil, fmt.Errorf("some error"))
		// 		mockfs.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(o.FS.Join)
		// 		mockfs.On("Lstat", mock.AnythingOfType("string")).Return(
		// 			func(filename string) os.FileInfo {
		// 				fi, _ := o.FS.Lstat(filename)
		// 				return fi
		// 			},
		// 			func(filename string) error {
		// 				_, err := o.FS.Lstat(filename)
		// 				return err
		// 			},
		// 		)
		// 		mockfs.On("Stat", mock.AnythingOfType("string")).Return(
		// 			func(filename string) os.FileInfo {
		// 				fi, _ := o.FS.Stat(filename)
		// 				return fi
		// 			},
		// 			func(filename string) error {
		// 				_, err := o.FS.Stat(filename)
		// 				return err
		// 			},
		// 		)
		// 		mockfs.On("ReadDir", mock.AnythingOfType("string")).Return(
		// 			func(filename string) []os.FileInfo {
		// 				fi, _ := o.FS.ReadDir(filename)
		// 				return fi
		// 			},
		// 			func(filename string) error {
		// 				_, err := o.FS.ReadDir(filename)
		// 				return err
		// 			},
		// 		)
		// 		return nil, fs.Create(mockfs), nil
		// 	},
		// },
		"Should fail when failed to delete project.yaml file": {
			projectName: "project",
			wantErr:     "failed to delete project 'project': " + os.ErrNotExist.Error(),
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll("kustomize/app1/overlays/project", 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(fmt.Errorf("some error"))
				return mockRepo, fs.Create(memfs), nil
			},
		},
		"Should fail when persist fails": {
			projectName: "project",
			wantErr:     "failed to push to repo: some error",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll("kustomize/app1/overlays/project", 0666)
				_ = billyUtils.WriteFile(memfs, "projects/project.yaml", []byte{}, 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(fmt.Errorf("some error"))
				return mockRepo, fs.Create(memfs), nil
			},
		},
		"Should remove entire app folder, if it contains only one overlay": {
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll("kustomize/app1/overlays/project", 0666)
				_ = billyUtils.WriteFile(memfs, "projects/project.yaml", []byte{}, 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.False(t, repofs.ExistsOrDie("kustomize/app1"))
			},
		},
		"Should remove only overlay, if app contains more overlays": {
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll("kustomize/app1/overlays/project", 0666)
				_ = memfs.MkdirAll("kustomize/app1/overlays/project2", 0666)
				_ = billyUtils.WriteFile(memfs, "projects/project.yaml", []byte{}, 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.True(t, repofs.ExistsOrDie("kustomize/app1/overlays"))
				assert.False(t, repofs.ExistsOrDie("kustomize/app1/overlays/project"))
			},
		},
		"Should handle multiple apps": {
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll("kustomize/app1/overlays/project", 0666)
				_ = memfs.MkdirAll("kustomize/app1/overlays/project2", 0666)
				_ = memfs.MkdirAll("kustomize/app2/overlays/project", 0666)
				_ = billyUtils.WriteFile(memfs, "projects/project.yaml", []byte{}, 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.True(t, repofs.ExistsOrDie("kustomize/app1/overlays"))
				assert.False(t, repofs.ExistsOrDie("kustomize/app1/overlays/project"))
				assert.False(t, repofs.ExistsOrDie("kustomize/app2"))
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
			opts := &BaseOptions{
				ProjectName: tt.projectName,
				FS:          fs.Create(memfs.New()),
			}
			if err := RunProjectDelete(context.Background(), opts); err != nil {
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
