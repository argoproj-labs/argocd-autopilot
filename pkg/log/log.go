package log

import (
	"context"

	"github.com/spf13/cobra"
)

var (
	G = GetLogger

	L Logger = NopLogger{}
)

type (
	loggerKey struct{}
	NopLogger struct{}
)

type Fields map[string]interface{}

type Logger interface {
	Printf(string, ...interface{})
	Debug(...interface{})
	Info(...interface{})
	Warn(...interface{})
	Fatal(...interface{})
	Error(...interface{})
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Fatalf(string, ...interface{})
	Errorf(string, ...interface{})

	WithField(string, interface{}) Logger
	WithFields(Fields) Logger
	WithError(error) Logger

	// AddPFlags adds persistent logger flags to cmd
	AddPFlags(*cobra.Command)
}

func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func GetLogger(ctx context.Context) Logger {
	logger := ctx.Value(loggerKey{})
	if logger == nil {
		if L == nil {
			panic("default logger not initialized")
		}
	}

	return logger.(Logger)
}

func (NopLogger) AddPFlags(*cobra.Command)               {}
func (NopLogger) Printf(string, ...interface{})          {}
func (NopLogger) Debug(...interface{})                   {}
func (NopLogger) Info(...interface{})                    {}
func (NopLogger) Warn(...interface{})                    {}
func (NopLogger) Fatal(...interface{})                   {}
func (NopLogger) Error(...interface{})                   {}
func (NopLogger) Debugf(string, ...interface{})          {}
func (NopLogger) Infof(string, ...interface{})           {}
func (NopLogger) Warnf(string, ...interface{})           {}
func (NopLogger) Fatalf(string, ...interface{})          {}
func (NopLogger) Errorf(string, ...interface{})          {}
func (l NopLogger) WithField(string, interface{}) Logger { return l }
func (l NopLogger) WithFields(Fields) Logger             { return l }
func (l NopLogger) WithError(error) Logger               { return l }
