package main

import (
	"context"
	"syscall"

	"github.com/argoproj/argocd-autopilot/cmd/gitops-agent/commands"
	"github.com/codefresh-io/pkg/helpers"
	"github.com/codefresh-io/pkg/log"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()
	lgr := log.FromLogrus(logrus.NewEntry(logrus.StandardLogger()), &log.LogrusConfig{Level: "info"})
	ctx = log.WithLogger(ctx, lgr)
	log.SetDefault(lgr)
	ctx = helpers.ContextWithCancelOnSignals(ctx, syscall.SIGINT, syscall.SIGTERM)

	cmd := commands.NewRoot()
	lgr.AddPFlags(cmd)

	defer func() {
		if err := recover(); err != nil {
			log.G(ctx).Fatal(err)
		}
	}()

	helpers.Die(cmd.ExecuteContext(ctx))
}
