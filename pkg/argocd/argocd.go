package argocd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-cd/v2/cmd/argocd/commands"
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

	cmd.Flags().AddFlagSet(addcmd.Flags())
	cmd.Flags().AddFlagSet(addcmd.InheritedFlags())

	return &addClusterImpl{root, args}, nil
}

func (a *addClusterImpl) Execute(ctx context.Context, clusterName string) error {
	a.cmd.SetArgs(append(a.args, clusterName))
	return a.cmd.ExecuteContext(ctx)
}
