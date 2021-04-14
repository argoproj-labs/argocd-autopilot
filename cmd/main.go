package main

import (
	"context"
	"syscall"

	"github.com/argoproj/argocd-autopilot/cmd/commands"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // used for authentication with cloud providers
)

func main() {
	ctx := context.Background()
	lgr := log.FromLogrus(logrus.NewEntry(logrus.New()), &log.LogrusConfig{Level: "info"})
	ctx = log.WithLogger(ctx, lgr)
	ctx = util.ContextWithCancelOnSignals(ctx, syscall.SIGINT, syscall.SIGTERM)

	c := commands.NewRoot()
	lgr.AddPFlags(c)

	if err := c.ExecuteContext(ctx); err != nil {
		log.G(ctx).Fatal(err)
	}
}
