package commands

import (
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/codefresh-io/pkg/helpers"
	"github.com/spf13/cobra"
)

func NewRoot() *cobra.Command {
	s := store.Get()

	cmd := &cobra.Command{
		Use:   s.BinaryName,
		Short: s.BinaryName + " <PUT SHORT DESCRIPTION HERE>",
		Run: func(cmd *cobra.Command, args []string) {
			helpers.Die(cmd.Help())
		},
		SilenceUsage:  true, // will not display usage when RunE returns an error
		SilenceErrors: true, // will not use fmt to print errors
	}

	cmd.AddCommand(NewStartCommand())
	cmd.AddCommand(s.NewVersionCommand())

	return cmd
}
