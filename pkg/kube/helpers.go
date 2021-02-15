package kube

import (
	"context"
	"errors"

	// "fmt"
	"os"
	"time"

	cferrors "github.com/codefresh-io/cf-argo/pkg/errors"
	"github.com/codefresh-io/cf-argo/pkg/log"

	fakeio "github.com/rhysd/go-fakeio"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/apply"
	kcmdutil "k8s.io/kubectl/pkg/cmd/util"
)

const (
	defaultPollInterval = time.Second * 2
	defaultPollTimeout  = time.Second * 5
)

func (c *client) apply(ctx context.Context, opts *ApplyOptions) error {
	if opts == nil {
		return cferrors.ErrNilOpts
	}

	if opts.Manifests == nil {
		return errors.New("no manifests")
	}

	applyWithTrack := ""
	applyWithStatus := false
	prune := false
	ios := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	o := apply.NewApplyOptions(ios)

	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a configuration to a resource in kubernetes",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.DeleteFlags.FileNameFlags.Filenames = &[]string{"-"}
			o.Overwrite = true
			o.Prune = prune
			o.PruneWhitelist = []string{
				"/v1/ConfigMap",
				"/v1/PersistentVolumeClaim",
				"/v1/Secret",
				"/v1/Service",
				"/v1/ServiceAccount",
				"apps/v1/DaemonSet",
				"apps/v1/Deployment",
				"batch/v1beta1/CronJob",
				// "networking/v1/Ingress",
			}

			if o.Namespace != "" {
				o.EnforceNamespace = true
			}

			err := o.Complete(c, cmd)
			if err != nil {
				return err
			}
			if opts.DryRun {
				o.DryRunStrategy = kcmdutil.DryRunClient
				outputFromat := "yaml"
				o.PrintFlags.OutputFormat = &outputFromat
			}

			fake := fakeio.StdinBytes([]byte{})
			defer fake.Restore()
			go func() {
				fake.StdinBytes(opts.Manifests)
				fake.CloseStdin()
			}()

			return o.Run()
		},
	}

	kcmdutil.AddDryRunFlag(applyCmd)
	kcmdutil.AddServerSideApplyFlags(applyCmd)
	kcmdutil.AddValidateFlags(applyCmd)
	kcmdutil.AddFieldManagerFlagVar(applyCmd, &o.FieldManager, apply.FieldManagerClientSideApply)

	applyCmd.Flags().BoolVar(&prune, "prune", false, "")
	applyCmd.Flags().BoolVar(&applyWithStatus, "status", false, "")
	applyCmd.Flags().StringVar(&applyWithTrack, "track", "ready", "")
	applyCmd.SetArgs([]string{})

	return applyCmd.Execute()
}

func (c *client) wait(ctx context.Context, opts *WaitOptions) error {
	if opts.DryRun {
		log.G(ctx).Debug("running in dry run mode, no wait")
		return nil
	}
	cs, err := c.KubernetesClientSet()
	if err != nil {
		return err
	}

	itr := 0

	rscs := map[*ResourceInfo]bool{}
	for _, r := range opts.Resources {
		rscs[r] = true
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
		for r := range rscs {
			ll := log.G(ctx).WithFields(log.Fields{
				"itr":       itr,
				"name":      r.Name,
				"namespace": r.Namespace,
			})
			ll.Debug("checking resource readiness")
			ready, err := r.Func(ctx, cs, r.Namespace, r.Name)
			if err != nil {
				ll.WithError(err).Debug("resource not ready")
				continue
			}
			if !ready {
				allReady = false
				ll.Debug("resource not ready")
				continue
			}

			ll.Debug("resource ready")
			delete(rscs, r)
		}

		return allReady, nil
	})
}
