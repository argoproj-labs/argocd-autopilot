package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/codefresh-io/cf-argo/cmd/root"
	"github.com/codefresh-io/cf-argo/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // used for authentication with cloud providers
)

func main() {
	logrusLogger := logrus.StandardLogger()
	ctx := context.Background()
	ctx = log.WithLogger(ctx, log.FromLogrus(logrus.NewEntry(logrusLogger)))
	ctx = contextWithCancel(ctx)

	logCfg := &log.Config{Level: "info"}

	c := root.New(ctx)
	c.PersistentFlags().AddFlagSet(logCfg.FlagSet())
	c.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return log.ConfigureLogrus(logrusLogger, logCfg)
	}

	if err := c.Execute(); err != nil {
		log.G(ctx).Fatal(err)
	}
}

func contextWithCancel(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		s := <-sig
		log.G(ctx).Debugf("got signal: %s", s)
		cancel()
	}()

	return ctx
}
