package kube

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/cmd/apply"
	del "k8s.io/kubectl/pkg/cmd/delete"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

//go:generate mockery --name Factory --filename kube.go

const (
	defaultPollInterval = time.Second * 2
	defaultPollTimeout  = time.Minute * 5
)

// WaitDeploymentReady can be used as a generic 'WaitFunc' for deployment.
func WaitDeploymentReady(ctx context.Context, f Factory, ns, name string) (bool, error) {
	cs, err := f.KubernetesClientSet()
	if err != nil {
		return false, err
	}

	d, err := cs.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	return d.Status.ReadyReplicas >= *d.Spec.Replicas, nil
}

type (
	Factory interface {
		// KubernetesClientSet returns a new kubernetes clientset or error
		KubernetesClientSet() (kubernetes.Interface, error)

		// KubernetesClientSetOrDie calls KubernetesClientSet() and panics if it returns an error
		KubernetesClientSetOrDie() kubernetes.Interface

		// ToRESTConfig returns a rest Config object or error
		ToRESTConfig() (*restclient.Config, error)

		// Apply applies the provided manifests on the specified namespace
		Apply(ctx context.Context, namespace string, manifests []byte) error

		Delete(ctx context.Context, resourceTypes []string, labelSelector string) error

		// Wait waits for all of the provided `Resources` to be ready by calling
		// the `WaitFunc` of each resource until all of them returns `true`
		Wait(context.Context, *WaitOptions) error
	}

	WaitFunc func(ctx context.Context, f Factory, ns, name string) (bool, error)

	Resource struct {
		Name      string
		Namespace string

		// WaitFunc will be called to check if the resources is ready. Should return (true, nil)
		// if the resources is ready, (false, nil) if the resource is not ready yet, or (false, err)
		// if some error occured (in that case the `Wait` will fail with that error).
		WaitFunc WaitFunc
	}

	WaitOptions struct {
		// Inverval the duration between each iteration of calling all of the resources' `WaitFunc`s.
		Interval time.Duration

		// Timeout the max time to wait for all of the resources to be ready. If not all of the
		// resourecs are ready at time this will cause `Wait` to return an error.
		Timeout time.Duration

		// Resources the list of resources to wait for.
		Resources []Resource
	}

	factory struct {
		f cmdutil.Factory
	}
)

func AddFlags(flags *pflag.FlagSet) Factory {
	confFlags := genericclioptions.NewConfigFlags(true)
	confFlags.AddFlags(flags)
	mvFlags := cmdutil.NewMatchVersionFlags(confFlags)

	return &factory{f: cmdutil.NewFactory(mvFlags)}
}

func DefaultIOStreams() genericclioptions.IOStreams {
	return genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}

// CurrentContext returns the name of the current kubernetes context or dies.
func CurrentContext() (string, error) {
	configAccess := clientcmd.NewDefaultPathOptions()
	conf, err := configAccess.GetStartingConfig()
	if err != nil {
		return "", err
	}

	return conf.CurrentContext, nil
}

func GenerateNamespace(namespace string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Annotations: map[string]string{
				"argocd.argoproj.io/sync-options": "Prune=false",
			},
		},
	}
}

func (f *factory) KubernetesClientSetOrDie() kubernetes.Interface {
	cs, err := f.KubernetesClientSet()
	util.Die(err)
	return cs
}

func (f *factory) KubernetesClientSet() (kubernetes.Interface, error) {
	return f.f.KubernetesClientSet()
}

func (f *factory) ToRESTConfig() (*restclient.Config, error) {
	return f.f.ToRESTConfig()
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			o.DeleteFlags.FileNameFlags.Filenames = &[]string{"-"}
			o.Overwrite = true

			if err := o.Complete(f.f, cmd); err != nil {
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

func (f *factory) Delete(ctx context.Context, resourceTypes []string, labelSelector string) error {
	o := &del.DeleteOptions{
		IOStreams:           DefaultIOStreams(),
		CascadingStrategy:   metav1.DeletePropagationForeground,
		DeleteAllNamespaces: true,
		LabelSelector:       labelSelector,
		WaitForDeletion:     true,
		Quiet:               true,
	}

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := strings.Join(resourceTypes, ",")
			if err := o.Complete(f.f, []string{args}, cmd); err != nil {
				return err
			}

			return o.RunDelete(f.f)
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmdutil.AddDryRunFlag(cmd)

	cmd.SetArgs([]string{})

	return cmd.ExecuteContext(ctx)
}

func (f *factory) Wait(ctx context.Context, opts *WaitOptions) error {
	itr := 0
	resources := map[*Resource]bool{}
	for i := range opts.Resources {
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
