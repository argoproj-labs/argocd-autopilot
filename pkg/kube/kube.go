package kube

import (
	"context"
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/util"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/cmd/apply"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type Factory interface {
	cmdutil.Factory
	KubernetesClientSetOrDie() *kubernetes.Clientset
	Apply(ctx context.Context, manifests []byte) error
}

type factory struct {
	cmdutil.Factory
}

func AddKubeConfigFlags(flags *pflag.FlagSet) (Factory, *genericclioptions.ConfigFlags) {
	confFlags := genericclioptions.NewConfigFlags(true)
	confFlags.AddFlags(flags)
	mvFlags := cmdutil.NewMatchVersionFlags(confFlags)

	return &factory{cmdutil.NewFactory(mvFlags)}, confFlags
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

func (f *factory) Apply(ctx context.Context, manifests []byte) error {
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

			if o.Namespace != "" {
				o.EnforceNamespace = true
			}

			if err := o.Complete(f, cmd); err != nil {
				return err
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
