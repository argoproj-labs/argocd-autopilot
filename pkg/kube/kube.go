package kube

import (
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/util"

	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type Factory interface {
	cmdutil.Factory
	KubernetesClientSetOrDie() *kubernetes.Clientset
}

type factory struct {
	cmdutil.Factory
}

func AddKubeConfigFlags(flags *pflag.FlagSet) Factory {
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
