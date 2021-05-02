package commands

import (
	"context"
	"fmt"

	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/spf13/cobra"
)

type (
	BaseOptions struct {
		CloneOptions *git.CloneOptions
		FS           fs.FS
		ProjectName  string
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
	cmd.Flags().StringVarP(&o.ProjectName, "project", "p", "", "Project name")
	return o, nil
}

func (o *BaseOptions) clone(ctx context.Context) (git.Repository, fs.FS, error) {
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
	r, filesystem, err := o.CloneOptions.Clone(ctx, o.FS)
	if err != nil {
		return nil, nil, err
	}

	log.G().Infof("using revision: \"%s\", installation path: \"%s\"", o.CloneOptions.Revision, filesystem.Root())
	if !filesystem.ExistsOrDie(store.Default.BootsrtrapDir) {
		return nil, nil, fmt.Errorf("Bootstrap folder not found, please execute `repo bootstrap --installation-path %s` command", filesystem.Root())
	}

	projExists := filesystem.ExistsOrDie(filesystem.Join(store.Default.ProjectsDir, o.ProjectName+".yaml"))
	if !projExists {
		return nil, nil, fmt.Errorf(util.Doc(fmt.Sprintf("project '%[1]s' not found, please execute `<BIN> project create %[1]s`", o.ProjectName)))
	}

	log.G().Debug("repository is ok")
	return r, filesystem, nil
}

var clone = func(ctx context.Context, cloneOpts *git.CloneOptions, filesystem fs.FS) (git.Repository, fs.FS, error) {
	return cloneOpts.Clone(ctx, filesystem)
}

var preRunE = func(ctx context.Context, filesystem fs.FS, opts *git.CloneOptions, projectName string) (git.Repository, fs.FS, error) {
	var (
		r   git.Repository
		err error
	)
	log.G().WithFields(log.Fields{
		"repoURL":  opts.URL,
		"revision": opts.Revision,
	}).Debug("starting with options: ")

	// clone repo
	log.G().Infof("cloning git repository: %s", opts.URL)
	r, filesystem, err = clone(ctx, opts, filesystem)
	if err != nil {
		return nil, nil, err
	}

	log.G().Infof("using revision: \"%s\", installation path: \"%s\"", opts.Revision, filesystem.Root())
	if !filesystem.ExistsOrDie(store.Default.BootsrtrapDir) {
		return nil, nil, fmt.Errorf("Bootstrap folder not found, please execute `repo bootstrap --installation-path %s` command", filesystem.Root())
	}

	projExists := filesystem.ExistsOrDie(filesystem.Join(store.Default.ProjectsDir, projectName+".yaml"))
	if !projExists {
		return nil, nil, fmt.Errorf(util.Doc(fmt.Sprintf("project '%[1]s' not found, please execute `<BIN> project create %[1]s`", projectName)))
	}

	log.G().Debug("repository is ok")
	return r, filesystem, nil
}

var die = util.Die
