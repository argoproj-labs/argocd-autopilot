package util

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/briandowns/spinner"
	billy "github.com/go-git/go-billy/v5"
)

const yamlSeperator = "\n---\n"

var (
	spinnerCharSet  = spinner.CharSets[26]
	spinnerDuration = time.Millisecond * 500
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

func WithSpinner(ctx context.Context, msg ...string) func() {
	if os.Getenv("NO_COLOR") != "" { // https://no-color.org/
		log.G(ctx).Info(msg)
		return func() {}
	}

	ctx, cancel := context.WithCancel(ctx)
	s := spinner.New(
		spinnerCharSet,
		spinnerDuration,
	)
	if len(msg) > 0 {
		s.Prefix = msg[0]
	}
	go func() {
		s.Start()
		<-ctx.Done()
		s.Stop()
		fmt.Println("")
	}()

	return func() {
		cancel()
		time.Sleep(time.Millisecond * 100)
	}
}

// Doc returns a string where the <BIN> is replaced with the binary name
func Doc(doc string) string {
	return strings.ReplaceAll(doc, "<BIN>", store.Get().BinaryName)
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

func JoinManifests(manifests ...[]byte) []byte {
	res := make([]string, 0, len(manifests))
	for _, m := range manifests {
		res = append(res, string(m))
	}
	return []byte(strings.Join(res, yamlSeperator))
}

func EnvValidateOrDie(fs billy.Filesystem, envName string) {
	if ok, err := Exists(fs, fs.Join(store.Common.EnvsDir, fmt.Sprintf("%s.yaml", envName))); err != nil {
		Die(err)
	} else if !ok {
		Die(fmt.Errorf("environment does not exist: %s", envName))
	}
}

func MustChroot(fs billy.Filesystem, path string) billy.Filesystem {
	newFS, err := fs.Chroot(path)
	Die(err)
	return newFS
}
