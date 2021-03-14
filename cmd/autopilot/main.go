package main

import (
	"context"
	"syscall"

	"github.com/argoproj/argocd-autopilot/cmd/autopilot/commands"
	"github.com/codefresh-io/pkg/helpers"
	"github.com/codefresh-io/pkg/log"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()
	lgr := log.FromLogrus(logrus.NewEntry(logrus.StandardLogger()), &log.LogrusConfig{Level: "info"})
	ctx = log.WithLogger(ctx, lgr)
	ctx = helpers.ContextWithCancelOnSignals(ctx, syscall.SIGINT, syscall.SIGTERM)

	c := commands.NewRoot(ctx)
	lgr.AddPFlags(c)

	defer func() {
		if err := recover(); err != nil {
			log.G(ctx).Fatal(err)
		}
	}()

	if err := c.Execute(); err != nil {
		log.G(ctx).Fatal(err)
	}
}
