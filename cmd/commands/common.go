package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	memfs "github.com/go-git/go-billy/v5/memfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/spf13/cobra"
)

type (
	BaseOptions struct {
		CloneOptions *git.CloneOptions
		FS           fs.FS
		ProjectName  string
	}
)

// used for mocking
var (
	die  = util.Die
	exit = os.Exit

	clone = func(ctx context.Context, cloneOpts *git.CloneOptions, filesystem fs.FS) (git.Repository, fs.FS, error) {
		return cloneOpts.Clone(ctx, filesystem)
	}

	prepareRepo = func(ctx context.Context, o *BaseOptions) (git.Repository, fs.FS, error) {
		var (
			r   git.Repository
			err error
		)
		log.G().WithFields(log.Fields{
			"repoURL":  o.CloneOptions.URL,
			"revision": o.CloneOptions.Revision,
		}).Debug("starting with options: ")

		// clone repo
		log.G().Infof("cloning git repository: %s", o.CloneOptions.URL)
		r, repofs, err := clone(ctx, o.CloneOptions, o.FS)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed cloning the repository: %w", err)
		}

		root := repofs.Root()
		log.G().Infof("using revision: \"%s\", installation path: \"%s\"", o.CloneOptions.Revision, root)
		if !repofs.ExistsOrDie(store.Default.BootsrtrapDir) {
			cmd := "repo bootstrap"
			if root != "/" {
				cmd += " --installation-path " + root
			}

			return nil, nil, fmt.Errorf("Bootstrap directory not found, please execute `%s` command", cmd)
		}

		if o.ProjectName != "" {
			projExists := repofs.ExistsOrDie(repofs.Join(store.Default.ProjectsDir, o.ProjectName+".yaml"))
			if !projExists {
				return nil, nil, fmt.Errorf(util.Doc(fmt.Sprintf("project '%[1]s' not found, please execute `<BIN> project create %[1]s`", o.ProjectName)))
			}
		}

		log.G().Debug("repository is ok")

		return r, repofs, nil
	}

	glob = func(fs fs.FS, pattern string) ([]string, error) {
		return billyUtils.Glob(fs, pattern)
	}
)

func addFlags(cmd *cobra.Command) (*BaseOptions, error) {
	cloneOptions, err := git.AddFlags(cmd)
	if err != nil {
		return nil, err
	}

	o := &BaseOptions{
		CloneOptions: cloneOptions,
		FS:           fs.Create(memfs.New()),
	}
	cmd.PersistentFlags().StringVarP(&o.ProjectName, "project", "p", "", "Project name")
	return o, nil
}
