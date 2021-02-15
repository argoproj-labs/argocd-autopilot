package errors

import (
	"context"
	"errors"

	"github.com/codefresh-io/cf-argo/pkg/log"
)

// Errors
var (
	ErrNilOpts = errors.New("options cannot be nil")
)

func MustContext(ctx context.Context, err error) {
	if err != nil {
		log.G(ctx).WithError(err).Fatal("must")
	}
}

func CheckErr(err error) {
	if err != nil {
		panic(err)
	}
}
