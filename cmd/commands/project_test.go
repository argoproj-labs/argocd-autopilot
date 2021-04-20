package commands

import (
	"context"
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/stretchr/testify/assert"
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
		repofs  fs.FS
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for ttName, tt := range tests {
		t.Run(ttName, func(t *testing.T) {
			got, err := getInstallationNamespace(tt.repofs)
			if (err != nil) != tt.wantErr {
				t.Errorf("getInstallationNamespace() error = %v, wantErr %v", err, tt.wantErr)
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
		"simple": {
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
