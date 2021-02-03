package root

import (
	"context"

	"github.com/codefresh-io/cf-argo/cmd/install"
	"github.com/codefresh-io/cf-argo/cmd/version"
	"github.com/codefresh-io/cf-argo/pkg/store"
	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	s := store.Get()

	cmd := &cobra.Command{
		Use:   s.BinaryName,
		Short: "cli tool for argo-enterprise solution",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true, // will not display usage when RunE returns an error
		SilenceErrors: true, // will not use fmt to print errors
	}

	cmd.AddCommand(version.New(ctx))
	cmd.AddCommand(install.New(ctx))

	return cmd
}
