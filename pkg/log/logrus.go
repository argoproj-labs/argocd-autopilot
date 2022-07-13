package log

import (
	"fmt"

	cmdutil "github.com/argoproj/argo-cd/v2/cmd/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	defaultLvl       = logrus.InfoLevel
	defaultFormatter = "text"
)

type LogrusFormatter string

type LogrusConfig struct {
	Level  string
	Format LogrusFormatter
}

type logrusAdapter struct {
	*logrus.Entry
	c *LogrusConfig
}

const (
	FormatterText LogrusFormatter = defaultFormatter
	FormatterJSON LogrusFormatter = "json"
)

func FromLogrus(l *logrus.Entry, c *LogrusConfig) Logger {
	if c == nil {
		c = &LogrusConfig{}
	}

	return &logrusAdapter{l, c}
}

func GetLogrusEntry(l Logger) (*logrus.Entry, error) {
	adpt, ok := l.(*logrusAdapter)
	if !ok {
		return nil, fmt.Errorf("not a logrus logger")
	}

	return adpt.Entry, nil
}

func (l *logrusAdapter) AddPFlags(cmd *cobra.Command) {
	flags := pflag.NewFlagSet("logrus", pflag.ContinueOnError)
	flags.StringVar(&l.c.Level, "log-level", l.c.Level, `set the log level, e.g. "debug", "info", "warn", "error"`)
	format := flags.String("log-format", defaultFormatter, `set the log format: "text", "json" (defaults to text)`)

	cmd.PersistentFlags().AddFlagSet(flags)
	cmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		switch *format {
		case string(FormatterJSON), string(FormatterText):
			l.c.Format = LogrusFormatter(*format)
		default:
			return fmt.Errorf("invalid log format: %s", *format)
		}

		cmdutil.LogFormat = *format
		cmdutil.LogLevel = l.c.Level

		return l.configure()
	}
}

func (l *logrusAdapter) Configure() error {
	return l.configure()
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

func (l *logrusAdapter) configure() error {
	var (
		err  error
		fmtr logrus.Formatter
		lvl  = defaultLvl
	)

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
