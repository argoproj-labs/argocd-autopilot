package log

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
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
}

type Config struct {
	Level string
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

func (c *Config) FlagSet() *pflag.FlagSet {
	flags := pflag.NewFlagSet("logrus", pflag.ContinueOnError)
	flags.StringVar(&c.Level, "log-level", c.Level, `set the log level, e.g. "debug", "info", "warn", "error"`)

	return flags
}

func ConfigureLogrus(logger *logrus.Logger, c *Config) error {
	if c.Level != "" {
		lvl, err := logrus.ParseLevel(c.Level)
		if err != nil {
			return err
		}

		logger.SetLevel(lvl)
	}
	return nil
}

type logrusAdapter struct {
	*logrus.Entry
}

func FromLogrus(l *logrus.Entry) Logger {
	return &logrusAdapter{l}
}

func (l *logrusAdapter) WithField(key string, val interface{}) Logger {
	return FromLogrus(l.Entry.WithField(key, val))
}

func (l *logrusAdapter) WithFields(fields Fields) Logger {
	return FromLogrus(l.Entry.WithFields(logrus.Fields(fields)))
}

func (l *logrusAdapter) WithError(err error) Logger {
	return FromLogrus(l.Entry.WithError(err))
}

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
