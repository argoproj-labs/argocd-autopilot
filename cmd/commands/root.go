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
		Use: s.BinaryName,
		Short: util.Doc(`<BIN> is used for installing and managing argo-cd installations and argo-cd
applications using gitops`),
		Long: util.Doc(`<BIN> is used for installing and managing argo-cd installations and argo-cd
applications using gitops.
		
Most of the commands in this CLI require you to specify a personal access token
for your git provider. This token is used to authenticate with your git provider
when performing operations on the gitops repository, such as cloning it and
pushing changes to it.

It is recommended that you export the $GIT_TOKEN and $GIT_REPO environment
variables in advanced to simplify the use of those commands.
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true, // will not display usage when RunE returns an error
		SilenceErrors: true, // will not use fmt to print errors
	}

	cmd.AddCommand(NewVersionCommand())
	cmd.AddCommand(NewRepoCommand())
	cmd.AddCommand(NewProjectCommand())
	cmd.AddCommand(NewAppCommand())

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
		if viper.IsSet(f.Name) && f.Value.String() == "" {
			die(cmd.Flags().Set(f.Name, viper.GetString(f.Name)))
		}
	})
	cmd.Flags().SortFlags = false
}
