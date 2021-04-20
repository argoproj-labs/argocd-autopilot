package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/fs/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRunProjectCreate(t *testing.T) {
	type args struct {
		ctx  context.Context
		opts *ProjectCreateOptions
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RunProjectCreate(tt.args.ctx, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("RunProjectCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getInstallationNamespace(t *testing.T) {
	tests := map[string]struct {
		nsYAML   string
		beforeFn func(m *mocks.FS, mockedFile *mocks.File)
		want     string
		wantErr  string
	}{
		"should return the namespace from namespace.yaml": {
			beforeFn: func(mockedFS *mocks.FS, mockedFile *mocks.File) {
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
			beforeFn: func(mockedFS *mocks.FS, _ *mocks.File) {
				mockedFS.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string"), "namespace.yaml").Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mockedFS.On("Open", mock.Anything).Return(nil, os.ErrNotExist)
			},
			wantErr: "file does not exist",
		},
		"should handle error during read": {
			beforeFn: func(mockedFS *mocks.FS, mockedFile *mocks.File) {
				mockedFS.On("Join", mock.AnythingOfType("string"), mock.AnythingOfType("string"), "namespace.yaml").Return(func(elem ...string) string {
					return strings.Join(elem, "/")
				})
				mockedFS.On("Open", mock.Anything).Return(mockedFile, nil)
				mockedFile.On("Read", mock.Anything).Return(0, fmt.Errorf("some error"))
			},
			wantErr: "failed to read namespace file: some error",
		},
		"should handle curropted namespace.yaml file": {
			beforeFn: func(mockedFS *mocks.FS, mockedFile *mocks.File) {
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
			mockedFile := &mocks.File{}
			mockedFS := &mocks.FS{}
			fs := fs.Create(mockedFS)
			tt.beforeFn(mockedFS, mockedFile)
			got, err := getInstallationNamespace(fs)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("getInstallationNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateProject(t *testing.T) {
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
			gotProject, gotAppSet := GenerateProject(tt.o)
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
