package utils

import (
	"context"

	"github.com/codefresh-io/cf-argo/pkg/log"
)

func MockLoggerContext() context.Context {
	return log.WithLogger(context.Background(), log.NopLogger{})
}
