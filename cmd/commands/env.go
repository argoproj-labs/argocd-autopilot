package commands

import (
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/ghodss/yaml"

	appset "github.com/argoproj-labs/applicationset/api/v1alpha1"
	v1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewEnvCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environments",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
			os.Exit(1)
		},
	}

	cmd.AddCommand(NewEnvCreateCommand())

	return cmd
}

func NewEnvCreateCommand() *cobra.Command {
	var (
		envName          string
		namespace        string
		repoURL          string
		revision         string
		installationPath string
		token            string
		dryRun           bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new environment",
		Example: util.Doc(`
`),
		Run: func(cmd *cobra.Command, args []string) {
			var (
				err error
			)

			// fs := memfs.New()
			// ctx := cmd.Context()

			log.G().WithFields(log.Fields{
				"env":      envName,
				"repoURL":  repoURL,
				"revision": revision,
			}).Debug("starting with options: ")

			env := generateAppSet(envName, namespace, repoURL, revision)

			envYAML, err := yaml.Marshal(env)
			util.Die(err)

			if dryRun {
				log.G().Printf("%s", envYAML)
				os.Exit(0)
			}
		},
	}
	//<env-name> --repo-url [--token|--secret] [--namespace] [--argocd-context] [--env-kube-context] [--dry-run]
	util.Die(viper.BindEnv("git-token", "GIT_TOKEN"))

	cmd.Flags().StringVar(&envName, "env", "", "Environment name")
	cmd.Flags().StringVar(&namespace, "namespace", "argocd", "Namespace")
	cmd.Flags().StringVar(&repoURL, "repo", "", "Repository URL")
	cmd.Flags().StringVar(&revision, "revision", "", "Repository branch")
	cmd.Flags().StringVar(&installationPath, "installation-path", "", "The path where we create the installation files (defaults to the root of the repository")
	cmd.Flags().StringVarP(&token, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "If true, print manifests instead of applying them to the cluster (nothing will be commited to git)")

	util.Die(cmd.MarkFlagRequired("env"))
	util.Die(cmd.MarkFlagRequired("repo"))
	util.Die(cmd.MarkFlagRequired("git-token"))

	return cmd
}

func generateAppSet(envName, namespace, repoURL, revision string) *appset.ApplicationSet {
	return &appset.ApplicationSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApplicationSet",
			APIVersion: appset.GroupVersion.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      envName,
			Namespace: namespace,
		},
		Spec: appset.ApplicationSetSpec{
			Generators: []appset.ApplicationSetGenerator{},
			Template: appset.ApplicationSetTemplate{
				ApplicationSetTemplateMeta: appset.ApplicationSetTemplateMeta{
					Name: "{{userGivenName}}",
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": store.Common.ManagedBy,
						"app.kubernetes.io/name":       "{{appName}}",
					},
				},
				Spec: v1alpha1.ApplicationSpec{
					Source: v1alpha1.ApplicationSource{
						RepoURL:        repoURL,
						TargetRevision: revision,
						Path:           "kustomize/components/{{appName}}/overlays/" + envName,
					},
					Destination: v1alpha1.ApplicationDestination{
						Server:    "https://kubernetes.default.svc",
						Namespace: namespace,
					},
					SyncPolicy: &v1alpha1.SyncPolicy{
						Automated: &v1alpha1.SyncPolicyAutomated{
							SelfHeal: true,
							Prune:    true,
						},
					},
				},
			},
		},
	}
}
