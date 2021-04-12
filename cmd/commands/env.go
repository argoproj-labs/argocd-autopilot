package commands

import (
	"fmt"
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/argocd"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/ghodss/yaml"
	memfs "github.com/go-git/go-billy/v5/memfs"

	appset "github.com/argoproj-labs/applicationset/api/v1alpha1"
	v1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewEnvCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "environment",
		Aliases: []string{"env"},
		Short:   "Manage environments",
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
		envName        string
		namespace      string
		envKubeContext string
		dryRun         bool
		addCmd         argocd.AddClusterCmd
		repoOpts       *git.CloneOptions
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new environment",
		Example: util.Doc(`
`),
		Run: func(cmd *cobra.Command, args []string) {
			var (
				err              error
				repoURL          = cmd.Flag("repo").Value.String()
				installationPath = cmd.Flag("installation-path").Value.String()
				revision         = cmd.Flag("revision").Value.String()
				namespace        = cmd.Flag("namespace").Value.String()
				fs               = memfs.New()
				ctx              = cmd.Context()
			)

			log.G().WithFields(log.Fields{
				"env":      envName,
				"repoURL":  repoURL,
				"revision": revision,
			}).Debug("starting with options: ")

			genPath := fs.Join(installationPath, "kustomize/{{appName}}/overlays", envName, "config.json")
			srcPath := fs.Join(installationPath, "kustomize/{{appName}}/overlays", envName)
			envYAML, err := generateAppSet(envName, namespace, repoURL, revision, genPath, srcPath, envKubeContext)
			util.Die(err)

			if dryRun {
				log.G().Printf("%s", envYAML)
				os.Exit(0)
			}

			bootstrapPath := fs.Join(installationPath, store.Common.BootsrtrapDir)

			log.G().Infof("cloning repo: %s", repoURL)

			// clone GitOps repo
			r, err := repoOpts.Clone(ctx, fs)
			util.Die(err)

			log.G().Infof("using revision: \"%s\", installation path: \"%s\"", revision, installationPath)
			exists, err := util.Exists(fs, bootstrapPath)
			util.Die(err)

			if !exists {
				util.Die(fmt.Errorf("Bootstrap folder not found, please execute `repo bootstrap --installation-path %s` command", installationPath))
			}

			log.G().Debug("repository is ok")

			if envKubeContext != "https://kubernetes.default.svc" {
				util.Die(addCmd.Execute(ctx, envKubeContext), "failed to add new cluster credentials")
			}

			envsPath := fs.Join(installationPath, store.Common.EnvsDir)
			writeFile(fs, fs.Join(envsPath, envName+".yaml"), envYAML)

			log.G().Infof("pushing new env manifest to repo")
			util.Die(r.Persist(ctx, &git.PushOptions{
				CommitMsg: "Added env " + envName,
			}))

			log.G().Infof("Done creating %s environment", envName)
		},
	}

	cmd.Flags().StringVar(&envName, "env", "", "Environment name")
	cmd.Flags().StringVar(&namespace, "namespace", "argocd", "Namespace")
	cmd.Flags().StringVar(&envKubeContext, "env-kube-context", "https://kubernetes.default.svc", "env kube context")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "If true, print manifests instead of applying them to the cluster (nothing will be commited to git)")

	addCmd, err := argocd.AddClusterAddFlags(cmd)
	util.Die(err)

	repoOpts, err = git.AddFlags(cmd)
	util.Die(err)

	util.Die(cmd.MarkFlagRequired("env"))

	return cmd
}

func generateAppSet(envName, namespace, repoURL, revision, genPath, srcPath, server string) ([]byte, error) {
	appSet := &appset.ApplicationSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApplicationSet",
			APIVersion: appset.GroupVersion.String(),
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
								Path: genPath,
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
						Path:           srcPath,
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
