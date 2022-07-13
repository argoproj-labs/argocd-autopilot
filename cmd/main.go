package main

import (
	"context"
	"syscall"

	"github.com/argoproj-labs/argocd-autopilot/cmd/commands"
	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	"github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // used for authentication with cloud providers
)

//go:generate sh -c "echo  generating command docs... && cd .. && ARGOCD_CONFIG_DIR=/home/user/.config/argocd go run ./hack/cmd-docs/main.go"

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
