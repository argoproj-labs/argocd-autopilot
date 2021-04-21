package commands

import (
	"context"

	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
)

var clone = func(ctx context.Context, cloneOpts *git.CloneOptions, filesystem fs.FS) (git.Repository, fs.FS, error) {
	return cloneOpts.Clone(ctx, filesystem)
}
