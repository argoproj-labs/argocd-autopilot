package kube

import (
	"bytes"
	"context"
	"html/template"
	"path/filepath"
	"time"

	"github.com/codefresh-io/cf-argo/pkg/log"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/util/homedir"
	kcmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type (
	Config struct {
		cfg *genericclioptions.ConfigFlags
	}

	Client interface {
		kcmdutil.Factory
		Apply(ctx context.Context, opts *ApplyOptions) error
		Delete(ctx context.Context, opts *DeleteOptions) error
		Wait(ctx context.Context, opts *WaitOptions) error
	}
	client struct {
		kcmdutil.Factory
		log log.Logger
	}

	ResourceInfo struct {
		Name      string
		Namespace string
		Func      func(ctx context.Context, c Client, ns, name string) (bool, error)
	}

	WaitOptions struct {
		Interval  time.Duration
		Timeout   time.Duration
		Resources []*ResourceInfo
		DryRun    bool
	}

	ApplyOptions struct {
		// IOStreams the std streams used by the apply command
		Manifests []byte

		// DryRunStrategy by default false, if true will be set to "client" dry-run modes, see kubectl apply --help
		DryRun bool
	}

	DeleteOptions struct {
		// IOStreams the std streams used by the apply command
		Manifests []byte

		// DryRunStrategy by default false, if true will be set to "client" dry-run modes, see kubectl apply --help
		DryRun bool
	}
)

func NewConfig() *Config {
	return &Config{genericclioptions.NewConfigFlags(true)}
}

func (c *Config) AddFlagSet(cmd *cobra.Command) {
	flags := pflag.NewFlagSet("kubernetes", pflag.ContinueOnError)

	flags.StringVar(c.cfg.KubeConfig, "kubeconfig", viper.GetString("kubeconfig"), "path to the kubeconfig file [KUBECONFIG] (default: ~/.kube/config)")
	flags.StringVar(c.cfg.Context, "kube-context", viper.GetString("kube-context"), "name of the kubeconfig context to use (default: current context)")
	viper.SetDefault("kubeconfig", defaultConfigPath())
	_ = viper.BindEnv("kubeconfig", "KUBECONFIG")

	cmd.Flags().AddFlagSet(flags)
}

func NewForConfig(ctx context.Context, cfg *Config) Client {
	l := log.G(ctx)
	if *cfg.cfg.Context != "" {
		l = l.WithField("context", *cfg.cfg.Context)
	}

	return &client{kcmdutil.NewFactory(kcmdutil.NewMatchVersionFlags(cfg.cfg)), l}
}

func KustBuild(path string, values interface{}) ([]byte, error) {
	kopts := krusty.MakeDefaultOptions()
	kopts.DoLegacyResourceSort = true

	k := krusty.MakeKustomizer(filesys.MakeFsOnDisk(), kopts)

	res, err := k.Run(path)
	if err != nil {
		return nil, err
	}

	data, err := res.AsYaml()
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("").Parse(string(data))
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(make([]byte, 0, 4096))

	err = tpl.Execute(buf, values)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (c *client) Apply(ctx context.Context, opts *ApplyOptions) error {
	return c.apply(ctx, opts)
}

func (c *client) Delete(ctx context.Context, opts *DeleteOptions) error {
	return c.delete(ctx, opts)
}

func (c *client) Wait(ctx context.Context, opts *WaitOptions) error {
	return c.wait(ctx, opts)
}

func defaultConfigPath() string {
	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}
