package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/application"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	"github.com/ghodss/yaml"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/spf13/cobra"
)

type (
	AppCreateOptions struct {
		EnvName      string
		FS           fs.FS
		AppOpts      *application.CreateOptions
		CloneOptions *git.CloneOptions
	}
)

func NewAppCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "application",
		Aliases: []string{"app"},
		Short:   "Manage applications",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
			os.Exit(1)
		},
	}

	cmd.AddCommand(NewAppCreateCommand())

	return cmd
}

func NewAppCreateCommand() *cobra.Command {
	var (
		envName   string
		appOpts   *application.CreateOptions
		cloneOpts *git.CloneOptions
	)

	cmd := &cobra.Command{
		Use:   "create [APP_NAME]",
		Short: "Create an application in an environment",
		Example: util.Doc(`
# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:
	
		export GIT_TOKEN=<token>
		export GIT_REPO=<repo_url>

# or with the flags:
	
		--token <token> --repo <repo_url>
		
# Create a new application from kustomization in a remote repository
	
	<BIN> app create <new_app_name> --app github.com/some_org/some_repo/manifests?ref=v1.2.3 --env env_name
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				log.G().Fatal("must enter application name")
			}

			appOpts.AppName = args[0]

			return RunAppCreate(cmd.Context(), &AppCreateOptions{
				EnvName:      envName,
				FS:           fs.Create(memfs.New()),
				AppOpts:      appOpts,
				CloneOptions: cloneOpts,
			})
		},
	}

	cmd.Flags().StringVar(&envName, "env", "", "Environment name")
	util.Die(cmd.MarkFlagRequired("env"))

	appOpts = application.AddFlags(cmd)
	util.Die(cmd.MarkFlagRequired("app"))
	cloneOpts, err := git.AddFlags(cmd)
	util.Die(err)

	return cmd
}

func RunAppCreate(ctx context.Context, opts *AppCreateOptions) error {
	log.G().WithFields(log.Fields{
		"repoURL":  opts.CloneOptions.URL,
		"revision": opts.CloneOptions.Revision,
		"appName":  opts.AppOpts.AppName,
	}).Debug("starting with options: ")

	// clone repo
	log.G().Infof("cloning git repository: %s", opts.CloneOptions.URL)
	r, err := opts.CloneOptions.Clone(ctx, opts.FS)
	if err != nil {
		return err
	}

	log.G().Infof("using installation path: %s", opts.CloneOptions.RepoRoot)
	opts.FS.ChrootOrDie(opts.CloneOptions.RepoRoot)

	if !opts.FS.ExistsOrDie(store.Default.BootsrtrapDir) {
		log.G().Fatalf("Bootstrap folder not found, please execute `repo bootstrap --installation-path %s` command", opts.CloneOptions.RepoRoot)
	}

	envExists := opts.FS.ExistsOrDie(opts.FS.Join(store.Default.EnvsDir, opts.EnvName+".yaml"))
	if !envExists {
		log.G().Fatalf(util.Doc(fmt.Sprintf("env '%[1]s' not found, please execute `<BIN> env create %[1]s`", opts.EnvName)))
	}

	log.G().Debug("repository is ok")

	app, err := opts.AppOpts.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse application from flags: %v", err)
	}

	// get application files
	basePath := opts.FS.Join(store.Default.KustomizeDir, opts.AppOpts.AppName, "base", "kustomization.yaml")
	baseYAML, err := yaml.Marshal(app.Base())
	util.Die(err, "failed to marshal app base kustomization")

	overlayPath := opts.FS.Join(store.Default.KustomizeDir, opts.AppOpts.AppName, "overlays", opts.EnvName, "kustomization.yaml")
	overlayYAML, err := yaml.Marshal(app.Overlay())
	util.Die(err, "failed to marshal app overlay kustomization")

	nsPath := opts.FS.Join(store.Default.KustomizeDir, opts.AppOpts.AppName, "overlays", opts.EnvName, "namespace.yaml")
	nsYAML, err := yaml.Marshal(app.Namespace())
	util.Die(err, "failed to marshal app overlay namespace")

	configJSONPath := opts.FS.Join(store.Default.KustomizeDir, opts.AppOpts.AppName, "overlays", opts.EnvName, "config.json")
	configJSON, err := json.Marshal(app.ConfigJson())
	util.Die(err, "failed to marshal app config.json")

	// Create Base
	log.G().Debugf("checking if application base already exists: '%s'", basePath)
	exists, err := opts.FS.CheckExistsOrWrite(basePath, baseYAML)
	util.Die(err, fmt.Sprintf("failed to create application base file at '%s'", basePath))
	if !exists {
		log.G().Infof("created application base file at '%s'", basePath)
	} else {
		log.G().Infof("application base file exists on '%s'", basePath)
	}

	// Create Overlay
	log.G().Debugf("checking if application overlay already exists: '%s'", overlayPath)
	exists, err = opts.FS.CheckExistsOrWrite(overlayPath, overlayYAML)
	util.Die(err, fmt.Sprintf("failed to create application overlay file at '%s'", basePath))
	if !exists {
		log.G().Infof("created application overlay file at '%s'", overlayPath)
	} else {
		// application already exists
		log.G().Infof("app '%s' already installed on env '%s'", opts.AppOpts.AppName, opts.EnvName)
		log.G().Infof("found overlay on '%s'", overlayPath)
		os.Exit(1)
	}

	exists, err = opts.FS.CheckExistsOrWrite(nsPath, nsYAML)
	util.Die(err, fmt.Sprintf("failed to create application namespace file at '%s'", basePath))
	if !exists {
		log.G().Infof("created application namespace file at '%s'", overlayPath)
	} else {
		// application already exists
		log.G().Infof("app '%s' already installed on env '%s'", opts.AppOpts.AppName, opts.EnvName)
		log.G().Infof("found overlay on '%s'", overlayPath)
		os.Exit(1)
	}

	// Create config.json
	exists, err = opts.FS.CheckExistsOrWrite(configJSONPath, configJSON)
	util.Die(err, fmt.Sprintf("failed to create application config file at '%s'", basePath))
	if !exists {
		log.G().Infof("created application config file at '%s'", configJSONPath)
	} else {
		log.G().Infof("application base file exists on '%s'", configJSONPath)
	}

	commitMsg := fmt.Sprintf("installed app \"%s\" on environment \"%s\"", opts.AppOpts.AppName, opts.EnvName)
	if opts.CloneOptions.RepoRoot != "" {
		commitMsg += fmt.Sprintf(" installation-path: %s", opts.CloneOptions.RepoRoot)
	}

	log.G().Info("committing changes to gitops repo...")
	util.Die(r.Persist(ctx, &git.PushOptions{CommitMsg: commitMsg}), "failed to push to repo")
	log.G().Info("application installed!")
	return nil
}
