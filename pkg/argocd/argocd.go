package argocd

import (
	"context"

	"github.com/argoproj-labs/argocd-autopilot/pkg/kube"
	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	// used to solve this issue: https://github.com/argoproj/argo-cd/issues/2907
	_ "github.com/argoproj-labs/argocd-autopilot/util/assets"

	"github.com/argoproj/argo-cd/v2/cmd/argocd/commands"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdcd "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	// AddClusterCmd when executed calls the 'argocd cluster add' command
	AddClusterCmd interface {
		Execute(ctx context.Context, clusterName string) error
	}

	addClusterImpl struct {
		cmd  *cobra.Command
		args []string
	}

	LoginOptions struct {
		Namespace string
		Username  string
		Password  string
	}
)

func AddClusterAddFlags(cmd *cobra.Command) (AddClusterCmd, error) {
	root := commands.NewCommand()
	args := []string{"cluster", "add"}
	addcmd, _, err := root.Find(args)
	if err != nil {
		return nil, err
	}

	fs, err := util.StealFlags(addcmd, []string{"logformat", "loglevel", "namespace"})
	if err != nil {
		return nil, err
	}

	cmd.Flags().AddFlagSet(fs)

	return &addClusterImpl{root, args}, nil
}

func CheckAppSynced(ctx context.Context, f kube.Factory, ns, name string) (bool, error) {
	rc, err := f.ToRESTConfig()
	if err != nil {
		return false, err
	}

	c, err := argocdcd.NewForConfig(rc)
	if err != nil {
		return false, err
	}

	app, err := c.ArgoprojV1alpha1().Applications(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		se, ok := err.(*errors.StatusError)
		if !ok || se.ErrStatus.Reason != metav1.StatusReasonNotFound {
			return false, err
		}

		return false, nil
	}

	log.G().Debugf("Application found, Sync Status = %s", app.Status.Sync.Status)
	return app.Status.Sync.Status == v1alpha1.SyncStatusCodeSynced, nil
}

func (a *addClusterImpl) Execute(ctx context.Context, clusterName string) error {
	a.cmd.SetArgs(append(a.args, clusterName))
	return a.cmd.ExecuteContext(ctx)
}

func Login(opts *LoginOptions) error {
	root := commands.NewCommand()
	args := []string{
		"login",
		"--port-forward",
		"--port-forward-namespace",
		opts.Namespace,
		"--password",
		opts.Password,
		"--username",
		opts.Username,
		"--name",
		"autopilot",
	}

	root.SetArgs(args)
	return root.Execute()
}
