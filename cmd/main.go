package main

import (
	"context"
	"syscall"

	"github.com/argoproj/argocd-autopilot/cmd/commands"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()
	lgr := log.FromLogrus(logrus.NewEntry(logrus.StandardLogger()), &log.LogrusConfig{Level: "info"})
	ctx = log.WithLogger(ctx, lgr)
	ctx = util.ContextWithCancelOnSignals(ctx, syscall.SIGINT, syscall.SIGTERM)

	c := commands.NewRoot()
	lgr.AddPFlags(c)

	defer func() {
		if err := recover(); err != nil {
			log.G(ctx).Fatal(err)
		}
	}()

	if err := c.ExecuteContext(ctx); err != nil {
		log.G(ctx).Fatal(err)
	}
}
