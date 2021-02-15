package log

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	defaultLvl       = logrus.InfoLevel
	defaultFormatter = "text"
)

type LogrusFormatter string

const (
	FormatterText LogrusFormatter = defaultFormatter
	FormatterJSON LogrusFormatter = "json"
)

type LogrusConfig struct {
	Level  string
	Format LogrusFormatter
}

type logrusAdapter struct {
	*logrus.Entry
	c *LogrusConfig
}

func FromLogrus(l *logrus.Entry, c *LogrusConfig) Logger {
	return &logrusAdapter{l, c}
}

func (l *logrusAdapter) AddPFlags(cmd *cobra.Command) {
	flags := pflag.NewFlagSet("logrus", pflag.ContinueOnError)
	flags.StringVar(&l.c.Level, "log-level", l.c.Level, `set the log level, e.g. "debug", "info", "warn", "error"`)
	format := flags.String("log-format", defaultFormatter, `set the log format: "text", "json" (defaults to text)`)

	cmd.PersistentFlags().AddFlagSet(flags)
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		switch *format {
		case string(FormatterJSON), string(FormatterText):
			l.c.Format = LogrusFormatter(*format)
		default:
			return fmt.Errorf("invalid log format: %s", *format)
		}

		return l.configure(flags)
	}
}

func (l *logrusAdapter) Printf(format string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Printf(fmt.Sprintf("%s\n", format), args...)
	} else {
		fmt.Println(format)
	}
}

func (l *logrusAdapter) WithField(key string, val interface{}) Logger {
	return FromLogrus(l.Entry.WithField(key, val), l.c)
}

func (l *logrusAdapter) WithFields(fields Fields) Logger {
	return FromLogrus(l.Entry.WithFields(logrus.Fields(fields)), l.c)
}

func (l *logrusAdapter) WithError(err error) Logger {
	return FromLogrus(l.Entry.WithError(err), l.c)
}

func (l *logrusAdapter) configure(f *pflag.FlagSet) error {
	var err error
	var fmtr logrus.Formatter
	lvl := defaultLvl

	if l.c.Level != "" {
		lvl, err = logrus.ParseLevel(l.c.Level)
		if err != nil {
			return err
		}

	}

	if lvl < logrus.DebugLevel {
		fmtr = &logrus.TextFormatter{
			DisableTimestamp:       true,
			DisableLevelTruncation: true,
		}
	} else {
		fmtr = &logrus.TextFormatter{
			FullTimestamp: true,
		}
	}

	if l.c.Format == FormatterJSON {
		fmtr = &logrus.JSONFormatter{}
	}

	l.Logger.SetLevel(lvl)
	l.Logger.SetFormatter(fmtr)

	return nil
}
