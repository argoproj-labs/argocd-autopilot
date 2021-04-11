package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/ghodss/yaml"
	memfs "github.com/go-git/go-billy/v5/memfs"

	appset "github.com/argoproj-labs/applicationset/api/v1alpha1"
	v1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	argocmds "github.com/argoproj/argo-cd/v2/cmd/argocd/commands"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
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
		repoURL          string
		revision         string
		installationPath string
		token            string
		namespace        string
		envKubeContext   string
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

			log.G().WithFields(log.Fields{
				"env":      envName,
				"repoURL":  repoURL,
				"revision": revision,
			}).Debug("starting with options: ")

			envYAML, err := generateAppSet(envName, namespace, repoURL, revision, envKubeContext)
			util.Die(err)

			ctx := cmd.Context()

			if envKubeContext != "https://kubernetes.default.svc" {
				util.Die(addCluster2(ctx, envKubeContext))
			}

			if dryRun {
				log.G().Printf("%s", envYAML)
				os.Exit(0)
			}

			fs := memfs.New()
			bootstrapPath := fs.Join(installationPath, store.Common.BootsrtrapDir)

			log.G().Infof("cloning repo: %s", repoURL)

			// clone GitOps repo
			r, err := git.Clone(ctx, &git.CloneOptions{
				URL:      repoURL,
				Revision: revision,
				Auth: &git.Auth{
					Username: "username",
					Password: token,
				},
				FS: fs,
			})
			util.Die(err)

			log.G().Infof("using revision: \"%s\", installation path: \"%s\"", revision, installationPath)
			exists, err := util.Exists(fs, bootstrapPath)
			util.Die(err)

			if !exists {
				util.Die(fmt.Errorf("Bootstrap folder not found, please execute `repo bootstrap --installation-path %s` command", installationPath))
			}

			log.G().Debug("repository is ok")

			envsPath := fs.Join(installationPath, store.Common.EnvsDir)
			writeFile(fs, fs.Join(envsPath, envName+".yaml"), envYAML)

			log.G().Infof("pushing new env manifest to repo")
			util.Die(r.Persist(ctx, &git.PushOptions{
				CommitMsg: "Added env " + envName,
			}))

			log.G().Infof("Done creating %s environment", envName)
		},
	}
	//<env-name> --repo-url [--token|--secret] [--namespace] [--argocd-context] [--env-kube-context] [--dry-run]
	util.Die(viper.BindEnv("git-token", "GIT_TOKEN"))

	cmd.Flags().StringVar(&envName, "env", "", "Environment name")
	cmd.Flags().StringVar(&repoURL, "repo", "", "Repository URL")
	cmd.Flags().StringVar(&revision, "revision", "", "Repository branch")
	cmd.Flags().StringVar(&installationPath, "installation-path", "", "The path where we create the installation files (defaults to the root of the repository")
	cmd.Flags().StringVarP(&token, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
	cmd.Flags().StringVar(&namespace, "namespace", "argocd", "Namespace")
	cmd.Flags().StringVar(&envKubeContext, "env-kube-context", "https://kubernetes.default.svc", "env kube context")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "If true, print manifests instead of applying them to the cluster (nothing will be commited to git)")

	util.Die(cmd.MarkFlagRequired("env"))
	util.Die(cmd.MarkFlagRequired("repo"))
	util.Die(cmd.MarkFlagRequired("git-token"))

	return cmd
}

func generateAppSet(envName, namespace, repoURL, revision, server string) ([]byte, error) {
	appSet := &appset.ApplicationSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApplicationSet",
			APIVersion: appset.GroupVersion.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      envName,
			Namespace: namespace,
		},
		Spec: appset.ApplicationSetSpec{
			Generators: []appset.ApplicationSetGenerator{
				{
					Git: &appset.GitGenerator{
						RepoURL:  repoURL,
						Revision: revision,
						Files: []appset.GitFileGeneratorItem{
							{
								Path: fmt.Sprintf("kustomize/**/overlays/%s/config.json", envName),
							},
						},
					},
				},
			},
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
						Path:           "kustomize/{{appName}}/overlays/" + envName,
					},
					Destination: v1alpha1.ApplicationDestination{
						Server:    server,
						Namespace: "{{namespace}}",
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
	return yaml.Marshal(appSet)
}

// func addCluster(ctx context.Context, clusterName string) error {
// 	client := apiclient.NewClientOrDie(&apiclient.ClientOptions{
// 		ConfigPath:           "/Users/noamgal/.argocd/config",
// 		PortForward:          true,
// 		PortForwardNamespace: "argocd",
// 	})
// 	_, cclient := client.NewClusterClientOrDie()
// 	ccr := &cluster.ClusterCreateRequest{
// 		Cluster: &v1alpha1v2.Cluster{
// 			Name: clusterName,
// 		},
// 	}
// 	cluster, err := cclient.Create(ctx, ccr)
// 	log.G().WithFields(log.Fields{
// 		"cluster": cluster,
// 		"error":   err,
// 	}).Info("Created cluster")
// 	return err
// }

func addCluster2(ctx context.Context, cluster string) error {
	cmd := argocmds.NewClusterAddCommand(&apiclient.ClientOptions{
		PortForward:          true,
		PortForwardNamespace: "argocd",
	}, clientcmd.NewDefaultPathOptions())
	cmd.SetArgs([]string{
		cluster,
	})
	err := cmd.ExecuteContext(ctx)
	return err
}
