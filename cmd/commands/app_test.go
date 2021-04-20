package commands

import (
	"testing"

	"github.com/argoproj/argocd-autopilot/pkg/application"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/stretchr/testify/assert"
)

func Test_getCommitMsg(t *testing.T) {
	tests := map[string]struct {
		opts     *AppCreateOptions
		assertFn func(t *testing.T, res string)
	}{
		"On root": {
			opts: &AppCreateOptions{
				CloneOptions: &git.CloneOptions{
					RepoRoot: "",
				},
				AppOpts: &application.CreateOptions{
					AppName: "foo",
				},
				ProjectName: "bar",
			},
			assertFn: func(t *testing.T, res string) {
				assert.Contains(t, res, "installed app 'foo' on project 'bar'")
				assert.NotContains(t, res, "installation-path")
			},
		},
		"On installation path": {
			opts: &AppCreateOptions{
				CloneOptions: &git.CloneOptions{
					RepoRoot: "foo/bar",
				},
				AppOpts: &application.CreateOptions{
					AppName: "foo",
				},
				ProjectName: "bar",
			},
			assertFn: func(t *testing.T, res string) {
				assert.Contains(t, res, "installed app 'foo' on project 'bar'")
				assert.Contains(t, res, "installation-path: 'foo/bar'")
			},
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			got := getCommitMsg(tt.opts)
			tt.assertFn(t, got)
		})
	}
}

func Test_writeApplicationFile(t *testing.T) {
	type args struct {
		repoFS fs.FS
		path   string
		name   string
		data   []byte
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := writeApplicationFile(tt.args.repoFS, tt.args.path, tt.args.name, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeApplicationFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("writeApplicationFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
