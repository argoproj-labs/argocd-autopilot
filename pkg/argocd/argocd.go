package argocd

import (
	"context"
	"fmt"
	"os"

	"github.com/argoproj-labs/argocd-autopilot/pkg/kube"
	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	"github.com/argoproj/argo-cd/v2/cmd/argocd/commands"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdcs "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/argoproj/gitops-engine/pkg/health"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
		Namespace  string
		Username   string
		Password   string
		KubeConfig string
		KubeContext string
		Insecure   bool
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

// GetAppSyncWaitFunc returns a WaitFunc that will return true when the Application
// is in Sync + Healthy state, and at the specific revision (if supplied. If revision is "", no revision check is made)
func GetAppSyncWaitFunc(revision string, waitForCreation bool) kube.WaitFunc {
	return func(ctx context.Context, f kube.Factory, ns, name string) (bool, error) {
		rc, err := f.ToRESTConfig()
		if err != nil {
			return false, err
		}

		c, err := argocdcs.NewForConfig(rc)
		if err != nil {
			return false, err
		}

		app, err := c.ArgoprojV1alpha1().Applications(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			se, ok := err.(*kerrors.StatusError)
			if !waitForCreation || !ok || se.ErrStatus.Reason != metav1.StatusReasonNotFound {
				return false, err
			}

			return false, nil
		}

		synced := app.Status.Sync.Status == v1alpha1.SyncStatusCodeSynced
		healthy := app.Status.Health.Status == health.HealthStatusHealthy
		onRevision := true
		if revision != "" {
			onRevision = revision == app.Status.Sync.Revision
		}

		log.G(ctx).Debugf("Application found, Sync Status: %s, Health Status: %s, Revision: %s", app.Status.Sync.Status, app.Status.Health.Status, app.Status.Sync.Revision)
		return synced && healthy && onRevision, nil
	}
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
		"--username",
		opts.Username,
		"--password",
		opts.Password,
		"--name",
		"autopilot",
	}

	if opts.KubeConfig != "" {
		origKubeConfig := os.Getenv("KUBECONFIG")
		defer func() { os.Setenv("KUBECONFIG", origKubeConfig) }()
		if err := os.Setenv("KUBECONFIG", opts.KubeConfig); err != nil {
			return fmt.Errorf("failed to set KUBECONFIG env var: %w", err)
		}
	}

	if opts.Insecure {
		args = append(args, "--plaintext")
	}


	root.SetArgs(args)
	return root.Execute()
}
