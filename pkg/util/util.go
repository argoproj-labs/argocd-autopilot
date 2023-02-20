package util

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	yamlSeperator = "\n---\n"
	indentation   = "    "
)

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

// KubeContextToServer returns the cluster server address for the provided kubernetes context
func KubeContextToServer(contextName string) (string, error) {
	configAccess := clientcmd.NewDefaultPathOptions()
	conf, err := configAccess.GetStartingConfig()
	if err != nil {
		return "", err
	}
	ctx := conf.Contexts[contextName]
	if ctx == nil {
		return "", fmt.Errorf("Context %s does not exist in kubeconfig", contextName)
	}
	cluster := conf.Clusters[ctx.Cluster]
	if cluster == nil {
		return "", fmt.Errorf("Cluster %s does not exist in kubeconfig", ctx.Cluster)
	}

	return cluster.Server, nil
}

// Die panics it the error is not nil. If a cause string is provided it will
// be displayed in the error message.
func Die(err error, cause ...string) {
	if err != nil {
		if len(cause) > 0 {
			panic(fmt.Errorf("%s: %w", cause[0], err))
		}
		panic(err)
	}
}

// WithSpinner create a spinner that prints a message and canceled if the
// given context is canceled or the returned stop function is called.
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
		// wait just enough time to prevent logs jumbling between spinner and main flow
		time.Sleep(time.Millisecond * 100)
	}
}

// Doc returns a string where all the '<BIN>' are replaced with the binary name
// and all the '\t' are replaced with a uniformed indentation using space.
func Doc(doc string) string {
	doc = strings.ReplaceAll(doc, "<BIN>", store.Get().BinaryName)
	doc = strings.ReplaceAll(doc, "\t", indentation)
	return doc
}

// MustParseDuration parses the given string as "time.Duration", or panic.
func MustParseDuration(dur string) time.Duration {
	d, err := time.ParseDuration(dur)
	Die(err)
	return d
}

// JoinManifests concats all of the provided yaml manifests with a yaml separator.
func JoinManifests(manifests ...[]byte) []byte {
	res := make([]string, 0, len(manifests))
	for _, m := range manifests {
		if m == nil {
			continue
		}
		res = append(res, string(m))
	}
	return []byte(strings.Join(res, yamlSeperator))
}

func SplitManifests(manifests []byte) [][]byte {
	str := string(manifests)
	stringManifests := strings.Split(str, yamlSeperator)
	res := make([][]byte, 0, len(stringManifests))
	for _, m := range stringManifests {
		res = append(res, []byte(m))
	}
	return res
}

func StealFlags(cmd *cobra.Command, exceptFor []string) (*pflag.FlagSet, error) {
	fs := &pflag.FlagSet{}
	ef := map[string]bool{}
	for _, e := range exceptFor {
		ef[e] = true
	}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if _, shouldSkip := ef[f.Name]; !shouldSkip {
			fs.AddFlag(f)
		}
	})

	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		if _, shouldSkip := ef[f.Name]; !shouldSkip {
			fs.AddFlag(f)
		}
	})

	return fs, nil
}

func CleanSliceWhiteSpaces(slc []string) []string {
	var res []string
	for i := range slc {
		if strings.TrimSpace(slc[i]) != "" {
			res = append(res, slc[i])
		}
	}
	
	return res
}
