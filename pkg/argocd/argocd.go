package argocd

import (
	"context"

	//"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-cd/v2/cmd/argocd/commands"
	//argocdclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
)

func AddCluster(ctx context.Context, cluster, namespace string) error {
	cmd := commands.NewCommand()
	cmd.SetArgs([]string{
		"cluster",
		"add",
		cluster,
	})
	_ = cmd.Flag("port-forward").Value.Set("true")
	_ = cmd.Flag("port-forward-namespace").Value.Set(namespace)
	return cmd.ExecuteContext(ctx)
}

func AddClusterAddFlags(cmd *cobra.Command) (*cobra.Command, error) {
	root := commands.NewCommand()
	addcmd, _, err := root.Find([]string{"cluster", "add"})
	if err != nil {
		return nil, err
	}

	cmd.Flags().AddFlagSet(addcmd.Flags())
	cmd.Flags().AddFlagSet(addcmd.InheritedFlags())
	return root, nil
}
