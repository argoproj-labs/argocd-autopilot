package commands

import (
	"github.com/spf13/cobra"

	server "github.com/argoproj/argocd-autopilot/server"
)

func NewStartCommand() *cobra.Command {
	var opts struct {
		port int
	}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "start the gitops-agent",
		Run: func(cmd *cobra.Command, args []string) {
			server.NewOrDie(cmd.Context(), &server.Options{
				Port: opts.port,
			}).Run()
		},
	}

	cmd.Flags().IntVar(&opts.port, "port", 8086, "server listen port")

	return cmd
}
