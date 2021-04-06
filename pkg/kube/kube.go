package kube

import (
	"context"
	"os"
	"time"

	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/cmd/apply"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

const (
	defaultPollInterval = time.Second * 2
	defaultPollTimeout  = time.Minute * 5
)

type Factory interface {
	cmdutil.Factory
	KubernetesClientSetOrDie() *kubernetes.Clientset
	Apply(ctx context.Context, namespace string, manifests []byte) error
	Wait(context.Context, *WaitOptions) error
}

type Resource struct {
	Name      string
	Namespace string
	WaitFunc  func(ctx context.Context, f Factory, ns, name string) (bool, error)
}

type WaitOptions struct {
	Interval  time.Duration
	Timeout   time.Duration
	Resources []Resource
}

type factory struct {
	cmdutil.Factory
}

func AddFlags(flags *pflag.FlagSet) Factory {
	confFlags := genericclioptions.NewConfigFlags(true)
	confFlags.AddFlags(flags)
	mvFlags := cmdutil.NewMatchVersionFlags(confFlags)

	return &factory{cmdutil.NewFactory(mvFlags)}
}

func DefaultIOStreams() genericclioptions.IOStreams {
	return genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}

func (f *factory) KubernetesClientSetOrDie() *kubernetes.Clientset {
	cs, err := f.KubernetesClientSet()
	util.Die(err)
	return cs
}

func (f *factory) Apply(ctx context.Context, namespace string, manifests []byte) error {
	reader, buf, err := os.Pipe()
	if err != nil {
		return err
	}

	o := apply.NewApplyOptions(DefaultIOStreams())

	stdin := os.Stdin
	os.Stdin = reader
	defer func() { os.Stdin = stdin }()

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			o.DeleteFlags.FileNameFlags.Filenames = &[]string{"-"}
			o.Overwrite = true

			if err := o.Complete(f, cmd); err != nil {
				return err
			}

			// order matters
			o.Namespace = namespace
			if o.Namespace != "" {
				o.EnforceNamespace = true
			}

			errc := make(chan error)
			go func() {
				if _, err = buf.Write(manifests); err != nil {
					errc <- err
				}
				if err = buf.Close(); err != nil {
					errc <- err
				}
				close(errc)
			}()

			if err = o.Run(); err != nil {
				return err
			}

			return <-errc
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmdutil.AddDryRunFlag(cmd)
	cmdutil.AddServerSideApplyFlags(cmd)
	cmdutil.AddValidateFlags(cmd)
	cmdutil.AddFieldManagerFlagVar(cmd, &o.FieldManager, apply.FieldManagerClientSideApply)

	cmd.SetArgs([]string{})

	return cmd.ExecuteContext(ctx)
}

func (f *factory) Wait(ctx context.Context, opts *WaitOptions) error {
	itr := 0
	resources := map[*Resource]bool{}
	for i, _ := range opts.Resources {
		resources[&opts.Resources[i]] = true
	}

	interval := defaultPollInterval
	timeout := defaultPollTimeout
	if opts.Interval > 0 {
		interval = opts.Interval
	}
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	return wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		itr += 1
		allReady := true

		for r := range resources {
			lgr := log.G().WithFields(log.Fields{
				"itr":       itr,
				"name":      r.Name,
				"namespace": r.Namespace,
			})

			lgr.Debug("checking resource readiness")
			ready, err := r.WaitFunc(ctx, f, r.Namespace, r.Name)
			if err != nil {
				lgr.WithError(err).Debug("resource not ready")
				continue
			}

			if !ready {
				allReady = false
				lgr.Debug("resource not ready")
				continue
			}

			lgr.Debug("resource ready")
			delete(resources, r)
		}

		return allReady, nil
	})
}
