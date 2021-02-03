package version

import (
	"context"
	"fmt"

	"github.com/codefresh-io/cf-argo/pkg/store"
	"github.com/spf13/cobra"
)

type options struct {
	long bool
}

func New(ctx context.Context) *cobra.Command {
	var opts options

	cmd := &cobra.Command{
		Use:   "version",
		Short: "show cli version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showVersion(&opts)
		},
	}

	cmd.Flags().BoolVar(&opts.long, "long", false, "display full version information")

	return cmd
}

func showVersion(opts *options) error {
	version := store.Get().Version
	if opts.long {
		fmt.Printf("Version: %s\n", version.Version)
		fmt.Printf("GitCommit: %s\n", version.GitCommit)
		fmt.Printf("GoVersion: %s\n", version.GoVersion)
		fmt.Printf("Platform: %s\n", version.Platform)
	} else {
		fmt.Printf("%+s\n", version.Version)
	}

	return nil
}
