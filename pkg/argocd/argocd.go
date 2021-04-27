package argocd

import (
	"context"

	"github.com/argoproj/argocd-autopilot/pkg/util"

	// used to solve this issue: https://github.com/argoproj/argo-cd/issues/2907
	_ "github.com/argoproj/argocd-autopilot/util/assets"

	"github.com/argoproj/argo-cd/v2/cmd/argocd/commands"
	"github.com/spf13/cobra"
)

type AddClusterCmd interface {
	Execute(ctx context.Context, clusterName string) error
}

type addClusterImpl struct {
	cmd  *cobra.Command
	args []string
}

func AddClusterAddFlags(cmd *cobra.Command) (AddClusterCmd, error) {
	root := commands.NewCommand()
	args := []string{"cluster", "add"}
	addcmd, _, err := root.Find(args)
	if err != nil {
		return nil, err
	}

	fs, err := util.StealFlags(addcmd, []string{"logformat", "loglevel", "namespace"})
	if err != nil {
		return nil, err
	}

	cmd.Flags().AddFlagSet(fs)

	return &addClusterImpl{root, args}, nil
}

func (a *addClusterImpl) Execute(ctx context.Context, clusterName string) error {
	a.cmd.SetArgs(append(a.args, clusterName))
	return a.cmd.ExecuteContext(ctx)
}
