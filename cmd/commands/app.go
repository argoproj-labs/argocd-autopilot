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
	appOpts = application.AddFlags(cmd)
	cloneOpts, err := git.AddFlags(cmd)
	util.Die(err)

	cmd.Flags().StringVar(&envName, "env", "", "Environment name")

	util.Die(cmd.MarkFlagRequired("env"))
	util.Die(cmd.MarkFlagRequired("app"))

	return cmd
}

func RunAppCreate(ctx context.Context, opts *AppCreateOptions) error {
	var (
		err error
		r   git.Repository
	)

	log.G().WithFields(log.Fields{
		"repoURL":  opts.CloneOptions.URL,
		"revision": opts.CloneOptions.Revision,
		"appName":  opts.AppOpts.AppName,
	}).Debug("starting with options: ")

	// clone repo
	log.G().Infof("cloning git repository: %s", opts.CloneOptions.URL)
	r, opts.FS, err = opts.CloneOptions.Clone(ctx, opts.FS)
	if err != nil {
		return err
	}
	log.G().Infof("using revision: \"%s\", installation path: \"%s\"", opts.CloneOptions.Revision, opts.FS.Root())

	if !opts.FS.ExistsOrDie(store.Default.BootsrtrapDir) {
		return fmt.Errorf(util.Doc("Bootstrap folder not found, please execute `<BIN> repo bootstrap --installation-path %s` command"), opts.FS.Root())
	}

	envExists := opts.FS.ExistsOrDie(opts.FS.Join(store.Default.EnvsDir, opts.EnvName+".yaml"))
	if !envExists {
		return fmt.Errorf(util.Doc("env '%[1]s' not found, please execute `<BIN> env create %[1]s`"), opts.EnvName)
	}
	log.G().Debug("repository is ok")

	app, err := opts.AppOpts.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse application from flags: %v", err)
	}

	if err = createApplicationFiles(opts.FS, app, opts.EnvName); err != nil {
		return err
	}

	log.G().Info("committing changes to gitops repo...")
	if err = r.Persist(ctx, &git.PushOptions{CommitMsg: getCommitMsg(opts)}); err != nil {
		return fmt.Errorf("failed to push to repo: %w", err)
	}
	log.G().Infof("installed application: %s", opts.AppOpts.AppName)

	return nil
}

func createApplicationFiles(repoFS fs.FS, app application.Application, env string) error {
	basePath := repoFS.Join(store.Default.KustomizeDir, app.Name(), "base")
	overlayPath := repoFS.Join(store.Default.KustomizeDir, app.Name(), "overlays", env)

	// get application files
	baseKustomizationPath := repoFS.Join(basePath, "kustomization.yaml")
	baseKustomizationYAML, err := yaml.Marshal(app.Base())
	if err != nil {
		return fmt.Errorf("failed to marshal app base kustomization: %w", err)
	}
	// get manifests - only used in flat installation mode
	manifestsPath := repoFS.Join(basePath, "install.yaml")
	manifests := app.Manifests()

	overlayKustomizationPath := repoFS.Join(overlayPath, "kustomization.yaml")
	overlayKustomizationYAML, err := yaml.Marshal(app.Overlay())
	if err != nil {
		return fmt.Errorf("failed to marshal app overlay kustomization: %w", err)
	}
	nsPath := repoFS.Join(overlayPath, "namespace.yaml")
	nsYAML, err := yaml.Marshal(app.Namespace())
	if err != nil {
		return fmt.Errorf("failed to marshal app overlay namespace: %w", err)
	}
	configJSONPath := repoFS.Join(overlayPath, "config.json")
	configJSON, err := json.Marshal(app.ConfigJson())
	if err != nil {
		return fmt.Errorf("failed to marshal app config.json: %w", err)
	}

	// Create Base
	if _, err = writeApplicationFile(repoFS, baseKustomizationPath, "base", baseKustomizationYAML); err != nil {
		return err
	}

	// Create Overlay
	if exists, err := writeApplicationFile(repoFS, overlayKustomizationPath, "overlay", overlayKustomizationYAML); err != nil {
		return err
	} else if exists {
		log.G().Infof("application %s already exists on environment: %s", app.Name(), env)
		os.Exit(1)
	}

	// Create application namespace file
	if _, err = writeApplicationFile(repoFS, nsPath, "application namespace", nsYAML); err != nil {
		return err
	}

	// Create config.json
	if _, err = writeApplicationFile(repoFS, configJSONPath, "config", configJSON); err != nil {
		return err
	}

	if manifests != nil {
		// flat installation mode
		if _, err = writeApplicationFile(repoFS, manifestsPath, "manifests", manifests); err != nil {
			return err
		}
	}

	return nil
}

func writeApplicationFile(repoFS fs.FS, path, name string, data []byte) (bool, error) {
	absPath := repoFS.Join(repoFS.Root(), path)
	exists, err := repoFS.CheckExistsOrWrite(path, data)
	if err != nil {
		return false, fmt.Errorf("failed to create %s file at '%s'", name, absPath)
	} else if exists {
		log.G().Infof("%s file exists on '%s'", name, absPath)
		return true, nil
	}
	log.G().Infof("created %s file at '%s'", name, absPath)
	return false, nil
}

func getCommitMsg(opts *AppCreateOptions) string {
	commitMsg := fmt.Sprintf("installed app \"%s\" on environment \"%s\"", opts.AppOpts.AppName, opts.EnvName)
	if opts.CloneOptions.RepoRoot != "" {
		commitMsg += fmt.Sprintf(" installation-path: %s", opts.FS.Root())
	}
	return commitMsg
}
