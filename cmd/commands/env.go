package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	appset "github.com/argoproj-labs/applicationset/api/v1alpha1"
	appsetv1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argocd-autopilot/pkg/argocd"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/ghodss/yaml"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var DefaultApplicationSetGeneratorInterval int64 = 20

type (
	EnvCreateOptions struct {
		EnvName        string
		Namespace      string
		EnvKubeContext string
		DryRun         bool
		FS             fs.FS
		CloneOptions   *git.CloneOptions
		AddCmd         argocd.AddClusterCmd
	}
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
		cloneOpts      *git.CloneOptions
	)

	cmd := &cobra.Command{
		Use:   "create [ENV]",
		Short: "Create a new environment",
		Example: util.Doc(`
# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:
	
		export GIT_TOKEN=<token>
		export GIT_REPO=<repo_url>

# or with the flags:
	
		--token <token> --repo <repo_url>
		
# Create a new environment
	
	<BIN> env create <new_env_name>

# Create a new environment in a specific path inside the GitOps repo

  <BIN> env create <new_env_name> --installation-path path/to/bootstrap/root
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				log.G().Fatal("must enter environment name")
			}
			envName = args[0]

			return RunEnvCreate(cmd.Context(), &EnvCreateOptions{
				EnvName:        envName,
				Namespace:      namespace,
				EnvKubeContext: envKubeContext,
				DryRun:         dryRun,
				FS:             fs.Create(memfs.New()),
				CloneOptions:   cloneOpts,
				AddCmd:         addCmd,
			})
		},
	}

	cmd.Flags().StringVar(&namespace, "namespace", "argocd", "The argo-cd namespace")
	cmd.Flags().StringVar(&envKubeContext, "env-kube-context", "", "The default destination kubernetes context for applications on this environment")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "If true, print manifests instead of applying them to the cluster (nothing will be commited to git)")

	addCmd, err := argocd.AddClusterAddFlags(cmd)
	util.Die(err)

	cloneOpts, err = git.AddFlags(cmd)
	util.Die(err)

	return cmd
}

func RunEnvCreate(ctx context.Context, opts *EnvCreateOptions) error {
	var (
		err error
		r   git.Repository
	)

	log.G().WithFields(log.Fields{
		"env":          opts.EnvName,
		"repoURL":      opts.CloneOptions.URL,
		"revision":     opts.CloneOptions.Revision,
		"installation": opts.CloneOptions.RepoRoot,
	}).Debug("starting with options: ")

	destServer := store.Default.DestServer
	if opts.EnvKubeContext != "" {
		destServer, err = util.KubeContextToServer(opts.EnvKubeContext)
		if err != nil {
			return err
		}
	}

	envApp := GenerateApplicationSet(&GenerateAppSetOptions{
		Name:              opts.EnvName,
		Namespace:         opts.Namespace,
		RepoURL:           opts.CloneOptions.URL,
		Revision:          opts.CloneOptions.Revision,
		InstallationPath:  opts.CloneOptions.RepoRoot,
		DefaultDestServer: destServer,
	})

	envAppYAML, err := yaml.Marshal(envApp)
	util.Die(err)

	if opts.DryRun {
		log.G().Printf("%s", envAppYAML)
		os.Exit(0)
	}

	log.G().Infof("cloning repo: %s", opts.CloneOptions.URL)

	// clone GitOps repo
	r, opts.FS, err = opts.CloneOptions.Clone(ctx, opts.FS)
	if err != nil {
		return err
	}

	log.G().Infof("using revision: \"%s\", installation path: \"%s\"", opts.CloneOptions.Revision, opts.FS.Root())
	if !opts.FS.ExistsOrDie(store.Default.BootsrtrapDir) {
		log.G().Fatalf("Bootstrap folder not found, please execute `repo bootstrap --installation-path %s` command", opts.FS.Root())
	}

	envExists := opts.FS.ExistsOrDie(opts.FS.Join(store.Default.EnvsDir, opts.EnvName+".yaml"))
	if envExists {
		log.G().Fatalf("env '%s' already exists", opts.EnvName)
	}

	log.G().Debug("repository is ok")

	if opts.EnvKubeContext != "" {
		log.G().Infof("adding cluster: %s", opts.EnvKubeContext)
		err = opts.AddCmd.Execute(ctx, opts.EnvKubeContext)
		if err != nil {
			return fmt.Errorf("failed to add new cluster credentials: %w", err)
		}
	}

	opts.FS.WriteFile(opts.FS.Join(store.Default.EnvsDir, opts.EnvName+".yaml"), envAppYAML)

	log.G().Infof("pushing new env manifest to repo")
	err = r.Persist(ctx, &git.PushOptions{
		CommitMsg: "Added env " + opts.EnvName,
	})
	if err != nil {
		return err
	}

	log.G().Infof("done creating %s environment", opts.EnvName)
	return nil
}

type GenerateAppSetOptions struct {
	Name              string
	Namespace         string
	DefaultDestServer string
	RepoURL           string
	Revision          string
	InstallationPath  string
}

func GenerateApplicationSet(o *GenerateAppSetOptions) *appset.ApplicationSet {
	return &appset.ApplicationSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApplicationSet",
			APIVersion: appset.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.Name,
			Namespace: o.Namespace,
		},
		Spec: appset.ApplicationSetSpec{
			Generators: []appset.ApplicationSetGenerator{
				{
					Git: &appset.GitGenerator{
						RepoURL:  o.RepoURL,
						Revision: o.Revision,
						Files: []appset.GitFileGeneratorItem{
							{
								Path: filepath.Join(o.InstallationPath, "kustomize", "**", "overlays", o.Name, "config.json"),
							},
						},
						Template: appset.ApplicationSetTemplate{
							Spec: appsetv1alpha1.ApplicationSpec{
								Destination: appsetv1alpha1.ApplicationDestination{
									Server:    o.DefaultDestServer,
									Namespace: "default",
								},
							},
						},
						RequeueAfterSeconds: &DefaultApplicationSetGeneratorInterval,
					},
				},
			},
			Template: appset.ApplicationSetTemplate{
				ApplicationSetTemplateMeta: appset.ApplicationSetTemplateMeta{
					Name: "{{userGivenName}}",
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": store.Default.ManagedBy,
						"app.kubernetes.io/name":       "{{appName}}",
					},
				},
				Spec: appsetv1alpha1.ApplicationSpec{
					Source: appsetv1alpha1.ApplicationSource{
						RepoURL:        o.RepoURL,
						TargetRevision: o.Revision,
						Path:           filepath.Join(o.InstallationPath, "kustomize", "{{appName}}", "overlays", o.Name),
					},
					Destination: appsetv1alpha1.ApplicationDestination{
						Server:    "{{destServer}}",
						Namespace: "{{destNamespace}}",
					},
					SyncPolicy: &appsetv1alpha1.SyncPolicy{
						Automated: &appsetv1alpha1.SyncPolicyAutomated{
							SelfHeal: true,
							Prune:    true,
						},
					},
				},
			},
		},
	}
}
