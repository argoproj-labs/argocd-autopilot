package argocd

import (
	"context"

	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	// used to solve this issue: https://github.com/argoproj/argo-cd/issues/2907
	_ "github.com/argoproj-labs/argocd-autopilot/util/assets"

	"github.com/argoproj/argo-cd/v2/cmd/argocd/commands"
	"github.com/spf13/cobra"
)

type (
	// AddClusterCmd when executed calls the 'argocd cluster add' command
	AddClusterCmd interface {
		Execute(ctx context.Context, clusterName string) error
	}

	LoginOptions struct {
		Namespace string
		Username  string
		Password  string
	}
)

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

func Login(opts *LoginOptions) error {
	root := commands.NewCommand()
	args := []string{
		"login",
		"--port-forward",
		"--port-forward-namespace",
		opts.Namespace,
		"--password",
		opts.Password,
		"--username",
		opts.Username,
		"--name",
		"autopilot",
	}

	root.SetArgs(args)
	return root.Execute()
}
