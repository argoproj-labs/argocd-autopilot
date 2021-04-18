package commands

import (
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/application"
	"github.com/argoproj/argocd-autopilot/pkg/argocd"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	"github.com/ghodss/yaml"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/spf13/cobra"
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
		Use:   "create [ENV]",
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

				fs  = fs.Create(memfs.New())
				ctx = cmd.Context()
			)

			if len(args) < 1 {
				log.G().Fatal("must enter environment name")
			}
			envName = args[0]

			log.G().WithFields(log.Fields{
				"env":          envName,
				"repoURL":      repoURL,
				"revision":     revision,
				"installation": installationPath,
			}).Debug("starting with options: ")

			destServer := application.DefaultDestServer
			if envKubeContext != "" {
				destServer, err = util.KubeContextToServer(envKubeContext)
				util.Die(err)
			}

			envApp := application.GenerateApplicationSet(&application.GenerateAppSetOptions{
				Name:              envName,
				Namespace:         namespace,
				RepoURL:           repoURL,
				Revision:          revision,
				InstallationPath:  installationPath,
				DefaultDestServer: destServer,
			})

			envAppYAML, err := yaml.Marshal(envApp)
			util.Die(err)

			if dryRun {
				log.G().Printf("%s", envAppYAML)
				os.Exit(0)
			}

			log.G().Infof("cloning repo: %s", repoURL)

			// clone GitOps repo
			r, err := repoOpts.Clone(ctx, fs)
			util.Die(err)

			fs.ChrootOrDie(installationPath)

			log.G().Infof("using installation path: %s", installationPath)
			if !fs.ExistsOrDie(store.Default.BootsrtrapDir) {
				log.G().Fatalf("Bootstrap folder not found, please execute `repo bootstrap --installation-path %s` command", installationPath)
			}

			envExists := fs.ExistsOrDie(fs.Join(store.Default.EnvsDir, envName+".yaml"))
			if envExists {
				log.G().Fatalf("env '%s' already exists", envName)
			}

			log.G().Debug("repository is ok")

			if envKubeContext != "" {
				log.G().Infof("adding cluster: %s", envKubeContext)
				util.Die(addCmd.Execute(ctx, envKubeContext), "failed to add new cluster credentials")
			}

			fs.WriteFile(fs.Join(store.Default.EnvsDir, envName+".yaml"), envAppYAML)

			log.G().Infof("pushing new env manifest to repo")
			util.Die(r.Persist(ctx, &git.PushOptions{
				CommitMsg: "Added env " + envName,
			}))

			log.G().Infof("done creating %s environment", envName)
		},
	}

	cmd.Flags().StringVar(&namespace, "namespace", "argocd", "The argo-cd namespace")
	cmd.Flags().StringVar(&envKubeContext, "env-kube-context", "", "The default destination kubernetes context for applications on this environment")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "If true, print manifests instead of applying them to the cluster (nothing will be commited to git)")

	addCmd, err := argocd.AddClusterAddFlags(cmd)
	util.Die(err)

	repoOpts, err = git.AddFlags(cmd)
	util.Die(err)

	return cmd
}
