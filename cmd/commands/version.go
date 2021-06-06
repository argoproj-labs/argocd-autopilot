package commands

import (
	"fmt"

	"github.com/argoproj-labs/argocd-autopilot/pkg/store"

	"github.com/spf13/cobra"
)

func NewVersionCommand() *cobra.Command {
	var opts struct {
		long bool
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show cli version",
		Run: func(_ *cobra.Command, _ []string) {
			s := store.Get()

			if opts.long {
				fmt.Printf("Version: %s\n", s.Version.Version)
				fmt.Printf("BuildDate: %s\n", s.Version.BuildDate)
				fmt.Printf("GitCommit: %s\n", s.Version.GitCommit)
				fmt.Printf("GoVersion: %s\n", s.Version.GoVersion)
				fmt.Printf("GoCompiler: %s\n", s.Version.GoCompiler)
				fmt.Printf("Platform: %s\n", s.Version.Platform)
			} else {
				fmt.Printf("%+s\n", s.Version.Version)
			}
		},
	}

	cmd.Flags().BoolVar(&opts.long, "long", false, "display full version information")

	return cmd
}
