package commands

import (
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/codefresh-io/pkg/log"
	"github.com/spf13/cobra"
)

func NewVersionCommand() *cobra.Command {
	var opts struct {
		long bool
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "show version",
		Run: func(cmd *cobra.Command, args []string) {
			version := store.Get().Version
			if opts.long {
				log.G().Printf("Version: %s", version.Version)
				log.G().Printf("GitCommit: %s", version.GitCommit)
				log.G().Printf("GoVersion: %s", version.GoVersion)
				log.G().Printf("Platform: %s", version.Platform)
			} else {
				log.G().Printf("%s", version.Version)
			}
		},
	}

	cmd.Flags().BoolVar(&opts.long, "long", false, "display full version information")

	return cmd
}
