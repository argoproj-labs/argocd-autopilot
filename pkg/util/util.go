package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/argoproj/argocd-autopilot/pkg/log"
	billy "github.com/go-git/go-billy/v5"
	"github.com/spf13/pflag"
)

// ContextWithCancelOnSignals returns a context that is canceled when one of the specified signals
// are received
func ContextWithCancelOnSignals(ctx context.Context, sigs ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, sigs...)

	go func() {
		cancels := 0
		for {
			s := <-sig
			cancels++
			if cancels == 1 {
				log.G(ctx).Printf("got signal: %s", s)
				cancel()
			} else {
				log.G(ctx).Printf("forcing exit")
				os.Exit(1)
			}
		}
	}()

	return ctx
}

// Die panics if err is not nil
func Die(err error, cause ...string) {
	if err != nil {
		if len(cause) > 0 {
			panic(fmt.Errorf("%s: %w", cause[0], err))
		}
		panic(err)
	}
}

func CopyDir(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		var relPath string = strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}

		absDst := filepath.Join(destination, relPath)
		if err = ensureDir(absDst); err != nil {
			return err
		}

		if info.IsDir() {
			err = os.Mkdir(absDst, info.Mode())
			if err != nil {
				if os.IsExist(err.(*os.PathError).Unwrap()) {
					return nil
				}
			}

			return err
		} else {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			return ioutil.WriteFile(absDst, data, info.Mode())
		}
	})
}

func ensureDir(path string) error {
	dstDir := filepath.Dir(path)
	if _, err := os.Stat(dstDir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		return os.MkdirAll(dstDir, 0755)
	}

	return nil
}

func Exists(fs billy.Filesystem, path string) (bool, error) {
	if _, err := fs.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}

		return false, nil
	}

	return true, nil
}

func MustGetString(flags *pflag.FlagSet, flag string) string {
	value, err := flags.GetString(flag)
	Die(err)

	return value
}

func MustGetBool(flags *pflag.FlagSet, flag string) bool {
	value, err := flags.GetBool(flag)
	Die(err)

	return value
}
