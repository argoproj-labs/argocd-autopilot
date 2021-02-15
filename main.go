package main

import (
	"context"
	"syscall"

	"github.com/codefresh-io/cf-argo/cmd/root"
	"github.com/codefresh-io/cf-argo/pkg/helpers"
	"github.com/codefresh-io/cf-argo/pkg/log"
	"github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // used for authentication with cloud providers
)

func main() {
	ctx := context.Background()
	lgr := log.FromLogrus(logrus.NewEntry(logrus.StandardLogger()), &log.LogrusConfig{Level: "info"})
	ctx = log.WithLogger(ctx, lgr)
	ctx = helpers.ContextWithCancelOnSignals(ctx, syscall.SIGINT, syscall.SIGTERM)

	c := root.New(ctx)
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
