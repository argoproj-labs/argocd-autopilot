package commands

import (
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func NewRoot() *cobra.Command {
	s := store.Get()

	cmd := &cobra.Command{
		Use:   s.BinaryName,
		Short: s.BinaryName + " is a cli tool for installing and managing argocd using gitops",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true, // will not display usage when RunE returns an error
		SilenceErrors: true, // will not use fmt to print errors
	}

	cmd.AddCommand(NewVersionCommand())
	cmd.AddCommand(NewRepoCommand())

	cobra.OnInitialize(func() { postInitCommands(cmd.Commands()) })

	return cmd
}

func postInitCommands(commands []*cobra.Command) {
	for _, cmd := range commands {
		presetRequiredFlags(cmd)
		if cmd.HasSubCommands() {
			postInitCommands(cmd.Commands())
		}
	}
}

func presetRequiredFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
			util.Die(cmd.Flags().Set(f.Name, viper.GetString(f.Name)))
		}
	})
	cmd.Flags().SortFlags = false
}
