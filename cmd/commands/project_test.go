package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/argoproj-labs/argocd-autopilot/pkg/application"
	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	fsmocks "github.com/argoproj-labs/argocd-autopilot/pkg/fs/mocks"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git"
	gitmocks "github.com/argoproj-labs/argocd-autopilot/pkg/git/mocks"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	appset "github.com/argoproj-labs/applicationset/api/v1alpha1"
	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/go-git/go-billy/v5/memfs"
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
				mockedRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{CommitMsg: "Added project 'project'"}).Return(fmt.Errorf("failed to persist"))
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
				mockedRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{CommitMsg: "Added project 'project'"}).Return(nil)
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
				CloneOpts:   &git.CloneOptions{},
				ProjectName: tt.projectName,
			}
			prepareRepo = func(_ context.Context, _ *git.CloneOptions, _ string) (git.Repository, fs.FS, error) {
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
		wantContextName        string
	}{
		"should generate project and appset with correct values": {
			o: &GenerateProjectOptions{
				Name:               "name",
				Namespace:          "namespace",
				DefaultDestServer:  "defaultDestServer",
				DefaultDestContext: "some-context-name",
				RepoURL:            "repoUrl",
				Revision:           "revision",
				InstallationPath:   "some/path",
			},
			wantName:               "name",
			wantNamespace:          "namespace",
			wantProjectDescription: "name project",
			wantRepoURL:            "repoUrl",
			wantRevision:           "revision",
			wantDefaultDestServer:  "defaultDestServer",
			wantContextName:        "some-context-name",
		},
	}
	for ttname, tt := range tests {
		t.Run(ttname, func(t *testing.T) {
			assert := assert.New(t)
			gotProject := &argocdv1alpha1.AppProject{}
			gotAppSet := &appset.ApplicationSet{}
			gotClusterResConf := &application.ClusterResConfig{}
			gotProjectYAML, gotAppSetYAML, _, gotClusterResConfigYAML, _ := generateProjectManifests(tt.o)
			assert.NoError(yaml.Unmarshal(gotProjectYAML, gotProject))
			assert.NoError(yaml.Unmarshal(gotAppSetYAML, gotAppSet))
			assert.NoError(yaml.Unmarshal(gotClusterResConfigYAML, gotClusterResConf))

			assert.Equal(tt.wantContextName, gotClusterResConf.Name)
			assert.Equal(tt.wantDefaultDestServer, gotClusterResConf.Server)

			assert.Equal(tt.wantName, gotProject.Name, "Project Name")
			assert.Equal(tt.wantNamespace, gotProject.Namespace, "Project Namespace")
			assert.Equal(tt.wantProjectDescription, gotProject.Spec.Description, "Project Description")
			assert.Equal(tt.o.DefaultDestServer, gotProject.Annotations[store.Default.DestServerAnnotation], "Application Set Default Destination Server")

			assert.Equal(tt.wantName, gotAppSet.Name, "Application Set Name")
			assert.Equal(tt.wantNamespace, gotAppSet.Namespace, "Application Set Namespace")
			assert.Equal(tt.wantRepoURL, gotAppSet.Spec.Generators[0].Git.RepoURL, "Application Set Repo URL")
			assert.Equal(tt.wantRevision, gotAppSet.Spec.Generators[0].Git.Revision, "Application Set Revision")

			assert.Equal(tt.wantNamespace, gotAppSet.Spec.Template.Namespace, "Application Set Template Namespace")
			assert.Equal(tt.wantName, gotAppSet.Spec.Template.Spec.Project, "Application Set Template Project")
		})
	}
}

func Test_getInstallationNamespace(t *testing.T) {
	tests := map[string]struct {
		nsYAML   string
		beforeFn func() fs.FS
		want     string
		wantErr  string
	}{
		"should return the namespace from namespace.yaml": {
			beforeFn: func() fs.FS {
				namespace := &argocdv1alpha1.Application{
					Spec: argocdv1alpha1.ApplicationSpec{
						Destination: argocdv1alpha1.ApplicationDestination{
							Namespace: "namespace",
						},
					},
				}
				repofs := fs.Create(memfs.New())
				_ = repofs.WriteYamls(filepath.Join(store.Default.BootsrtrapDir, store.Default.ArgoCDName+".yaml"), namespace)
				return repofs
			},
			want: "namespace",
		},
		"should handle file not found": {
			beforeFn: func() fs.FS {
				return fs.Create(memfs.New())
			},
			wantErr: "failed to unmarshal namespace: file does not exist",
		},
		"should handle error during read": {
			beforeFn: func() fs.FS {
				mfs := &fsmocks.FS{}
				mfs.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mfs.On("ReadYamls", filepath.Join(store.Default.BootsrtrapDir, store.Default.ArgoCDName+".yaml"), mock.Anything).Return(nil, fmt.Errorf("some error"))
				return mfs
			},
			wantErr: "failed to read namespace file: some error",
		},
		"should handle corrupted namespace.yaml file": {
			beforeFn: func() fs.FS {
				repofs := fs.Create(memfs.New())
				_ = billyUtils.WriteFile(repofs, filepath.Join(store.Default.BootsrtrapDir, store.Default.ArgoCDName+".yaml"), []byte("some string"), 0666)
				return repofs
			},
			wantErr: "failed to unmarshal namespace: error unmarshaling JSON: json: cannot unmarshal string into Go value of type v1alpha1.Application",
		},
	}
	for ttName, tt := range tests {
		t.Run(ttName, func(t *testing.T) {
			repofs := tt.beforeFn()
			got, err := getInstallationNamespace(repofs)
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
		beforeFn func(repofs fs.FS)
	}{
		"should return error if project file doesn't exist": {
			name:    "prod.yaml",
			wantErr: os.ErrNotExist.Error(),
		},
		"should failed when 2 files not found": {
			name:    "prod.yaml",
			wantErr: "expected at least 2 manifests when reading 'prod.yaml'",
			beforeFn: func(repofs fs.FS) {
				_ = billyUtils.WriteFile(repofs, "prod.yaml", []byte("content"), 0666)
			},
		},
		"should return AppProject": {
			name: "prod.yaml",
			beforeFn: func(repofs fs.FS) {
				appProj := argocdv1alpha1.AppProject{
					ObjectMeta: v1.ObjectMeta{
						Name:      "prod",
						Namespace: "ns",
					},
				}
				appSet := appset.ApplicationSet{}
				_ = repofs.WriteYamls("prod.yaml", appProj, appSet)
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
				tt.beforeFn(repofs)
			}

			got, _, err := getProjectInfoFromFile(repofs, tt.name)
			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("getProjectInfoFromFile() error = %v", err)
				}

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
		prepareRepo            func(ctx context.Context, cloneOpts *git.CloneOptions, projectName string) (git.Repository, fs.FS, error)
		getProjectInfoFromFile func(repofs fs.FS, name string) (*argocdv1alpha1.AppProject, *appset.ApplicationSet, error)
		assertFn               func(t *testing.T, out io.Writer)
	}{
		"should print to table": {
			opts: &ProjectListOptions{
				CloneOpts: &git.CloneOptions{},
				Out:       &bytes.Buffer{},
			},
			prepareRepo: func(_ context.Context, _ *git.CloneOptions, _ string) (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = billyUtils.WriteFile(memfs, "projects/prod.yaml", []byte{}, 0666)
				return nil, fs.Create(memfs), nil
			},
			getProjectInfoFromFile: func(_ fs.FS, _ string) (*argocdv1alpha1.AppProject, *appset.ApplicationSet, error) {
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
				assert.Contains(t, str, "NAME  NAMESPACE  DEFAULT CLUSTER  \n")
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
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app1", store.Default.OverlaysDir, "project"), 0666)
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
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app1", store.Default.OverlaysDir, "project"), 0666)
				_ = billyUtils.WriteFile(memfs, filepath.Join(store.Default.ProjectsDir, "project.yaml"), []byte{}, 0666)
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
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app1", store.Default.OverlaysDir, "project"), 0666)
				_ = billyUtils.WriteFile(memfs, filepath.Join(store.Default.ProjectsDir, "project.yaml"), []byte{}, 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app1")))
			},
		},
		"Should remove only overlay, if app contains more overlays": {
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app1", store.Default.OverlaysDir, "project"), 0666)
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app1", store.Default.OverlaysDir, "project2"), 0666)
				_ = billyUtils.WriteFile(memfs, filepath.Join(store.Default.ProjectsDir, "project.yaml"), []byte{}, 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.True(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app1", store.Default.OverlaysDir)))
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app1", store.Default.OverlaysDir, "project")))
			},
		},
		"Should remove directory apps": {
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app1", "project"), 0666)
				_ = billyUtils.WriteFile(memfs, filepath.Join(store.Default.ProjectsDir, "project.yaml"), []byte{}, 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app1")))
			},
		},
		"Should handle multiple apps": {
			projectName: "project",
			prepareRepo: func() (git.Repository, fs.FS, error) {
				memfs := memfs.New()
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app1", store.Default.OverlaysDir, "project"), 0666)
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app1", store.Default.OverlaysDir, "project2"), 0666)
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app2", store.Default.OverlaysDir, "project"), 0666)
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app3", "project"), 0666)
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app4", "project"), 0666)
				_ = memfs.MkdirAll(filepath.Join(store.Default.AppsDir, "app4", "project3"), 0666)
				_ = billyUtils.WriteFile(memfs, filepath.Join(store.Default.ProjectsDir, "project.yaml"), []byte{}, 0666)
				mockRepo := &gitmocks.Repository{}
				mockRepo.On("Persist", mock.AnythingOfType("*context.emptyCtx"), &git.PushOptions{
					CommitMsg: "Deleted project 'project'",
				}).Return(nil)
				return mockRepo, fs.Create(memfs), nil
			},
			assertFn: func(t *testing.T, repo git.Repository, repofs fs.FS) {
				repo.(*gitmocks.Repository).AssertExpectations(t)
				assert.True(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app1", store.Default.OverlaysDir)))
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app1", store.Default.OverlaysDir, "project")))
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app2")))
				assert.False(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app3")))
				assert.True(t, repofs.ExistsOrDie(filepath.Join(store.Default.AppsDir, "app4")))
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
				repo, repofs, err = tt.prepareRepo()
				return repo, repofs, err
			}
			opts := &ProjectDeleteOptions{
				ProjectName: tt.projectName,
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
