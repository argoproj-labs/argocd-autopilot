package log

import (
	"context"

	"github.com/spf13/cobra"
)

var (
	G = GetLogger

	L Logger = nopLogger{}
)

type (
	loggerKey struct{}
	nopLogger struct{}
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

func (nopLogger) AddPFlags(*cobra.Command)               {}
func (nopLogger) Printf(string, ...interface{})          {}
func (nopLogger) Debug(...interface{})                   {}
func (nopLogger) Info(...interface{})                    {}
func (nopLogger) Warn(...interface{})                    {}
func (nopLogger) Fatal(...interface{})                   {}
func (nopLogger) Error(...interface{})                   {}
func (nopLogger) Debugf(string, ...interface{})          {}
func (nopLogger) Infof(string, ...interface{})           {}
func (nopLogger) Warnf(string, ...interface{})           {}
func (nopLogger) Fatalf(string, ...interface{})          {}
func (nopLogger) Errorf(string, ...interface{})          {}
func (l nopLogger) WithField(string, interface{}) Logger { return l }
func (l nopLogger) WithFields(Fields) Logger             { return l }
func (l nopLogger) WithError(error) Logger               { return l }
