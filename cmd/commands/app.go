package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/argoproj/argocd-autopilot/pkg/application"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	memfs "github.com/go-git/go-billy/v5/memfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/spf13/cobra"
)

type (
	AppCreateOptions struct {
		BaseOptions
		AppOpts *application.CreateOptions
	}

	AppDeleteOptions struct {
		BaseOptions
		AppName string
		Global  bool
	}
)

func NewAppCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "application",
		Aliases: []string{"app"},
		Short:   "Manage applications",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
			exit(1)
		},
	}
	opts, err := addFlags(cmd)
	die(err)

	cmd.AddCommand(NewAppCreateCommand(opts))
	cmd.AddCommand(NewAppListCommand(opts))
	cmd.AddCommand(NewAppDeleteCommand(opts))

	return cmd
}

func NewAppCreateCommand(opts *BaseOptions) *cobra.Command {
	var (
		appOpts *application.CreateOptions
	)

	cmd := &cobra.Command{
		Use:   "create [APP_NAME]",
		Short: "Create an application in a specific project",
		Example: util.Doc(`
# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:

		export GIT_TOKEN=<token>
		export GIT_REPO=<repo_url>

# or with the flags:

		--git-token <token> --repo <repo_url>

# using the --type flag (kustomize|directory) is optional. If it is ommitted, <BIN> will clone
# the --app repository, and infer the type automatically.

# Create a new application from kustomization in a remote repository

	<BIN> app create <new_app_name> --app github.com/some_org/some_repo/manifests?ref=v1.2.3 --project project_name
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				log.G().Fatal("must enter application name")
			}

			appOpts.AppName = args[0]

			return RunAppCreate(cmd.Context(), &AppCreateOptions{
				BaseOptions: *opts,
				AppOpts:     appOpts,
			})
		},
	}
	appOpts = application.AddFlags(cmd)

	die(cmd.MarkFlagRequired("app"))

	return cmd
}

func RunAppCreate(ctx context.Context, opts *AppCreateOptions) error {
	r, repofs, err := prepareRepo(ctx, &opts.BaseOptions)
	if err != nil {
		return err
	}

	err = setAppOptsDefaults(ctx, repofs, opts)
	if err != nil {
		return err
	}

	app, err := opts.AppOpts.Parse(opts.CloneOptions, opts.ProjectName)
	if err != nil {
		return fmt.Errorf("failed to parse application from flags: %v", err)
	}

	if err = app.CreateFiles(repofs, opts.ProjectName); err != nil {
		if errors.Is(err, application.ErrAppAlreadyInstalledOnProject) {
			return fmt.Errorf("application '%s' already exists in project '%s': %w", app.Name(), opts.ProjectName, err)
		}

		return err
	}

	log.G().Info("committing changes to gitops repo...")
	if err = r.Persist(ctx, &git.PushOptions{CommitMsg: getCommitMsg(opts, repofs)}); err != nil {
		return fmt.Errorf("failed to push to repo: %w", err)
	}

	log.G().Infof("installed application: %s", opts.AppOpts.AppName)

	return nil
}

func setAppOptsDefaults(ctx context.Context, repofs fs.FS, opts *AppCreateOptions) error {
	var err error

	if opts.AppOpts.DestServer == store.Default.DestServer {
		opts.AppOpts.DestServer, err = getProjectDestServer(repofs, opts.ProjectName)
		if err != nil {
			return err
		}
	}

	if opts.AppOpts.DestNamespace == "" {
		opts.AppOpts.DestNamespace = "default"
	}

	if opts.AppOpts.AppType == "" {
		host, orgRepo, p, gitRef, _, _, _ := util.ParseGitUrl(opts.AppOpts.AppSpecifier)
		url := host + orgRepo
		log.G().Infof("Cloning repo: '%s', to infer app type from path '%s'", url, p)
		cloneOpts := &git.CloneOptions{
			URL:      url,
			Revision: gitRef,
			RepoRoot: p,
			Auth:     opts.CloneOptions.Auth,
		}
		_, repofs, err := cloneOpts.Clone(ctx, fs.Create(memfs.New()))
		if err != nil {
			return err
		}

		opts.AppOpts.AppType = application.InferAppType(repofs)
	}

	return nil
}

var getProjectDestServer = func(repofs fs.FS, projectName string) (string, error) {
	path := repofs.Join(store.Default.ProjectsDir, projectName+".yaml")
	p := &argocdv1alpha1.AppProject{}
	if err := repofs.ReadYamls(path, p); err != nil {
		return "", fmt.Errorf("failed to unmarshal project: %w", err)
	}

	return p.Annotations[store.Default.DestServerAnnotation], nil
}

func getCommitMsg(opts *AppCreateOptions, repofs fs.FS) string {
	commitMsg := fmt.Sprintf("installed app '%s' on project '%s'", opts.AppOpts.AppName, opts.ProjectName)
	if repofs.Root() != "" {
		commitMsg += fmt.Sprintf(" installation-path: '%s'", repofs.Root())
	}

	return commitMsg
}

func NewAppListCommand(opts *BaseOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [PROJECT_NAME]",
		Short: "List all applications in a project",
		Example: util.Doc(`
# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:

		export GIT_TOKEN=<token>
		export GIT_REPO=<repo_url>

# or with the flags:

		--git-token <token> --repo <repo_url>

# Get list of installed applications in a specifc project

	<BIN> app list <project_name>
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				log.G().Fatal("must enter a project name")
			}
			opts.ProjectName = args[0]

			return RunAppList(cmd.Context(), opts)
		},
	}

	return cmd
}

func RunAppList(ctx context.Context, opts *BaseOptions) error {
	_, repofs, err := prepareRepo(ctx, opts)
	if err != nil {
		return err
	}

	// get all apps beneath kustomize <project>\overlayes
	matches, err := billyUtils.Glob(repofs, repofs.Join(store.Default.AppsDir, "*", store.Default.OverlaysDir, opts.ProjectName))
	if err != nil {
		log.G().Fatalf("failed to run glob on %s", opts.ProjectName)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "PROJECT\tNAME\tDEST_NAMESPACE\tDEST_SERVER\t\n")

	for _, appPath := range matches {
		conf, err := getConfigFileFromPath(repofs, appPath)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", opts.ProjectName, conf.UserGivenName, conf.DestNamespace, conf.DestServer)
	}

	_ = w.Flush()
	return nil
}

func getConfigFileFromPath(repofs fs.FS, appPath string) (*application.Config, error) {
	path := repofs.Join(appPath, "config.json")
	b, err := repofs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s'", path)
	}

	conf := application.Config{}
	err = json.Unmarshal(b, &conf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal file '%s'", path)
	}

	return &conf, nil
}

func NewAppDeleteCommand(opts *BaseOptions) *cobra.Command {
	var (
		appName string
		global  bool
	)

	cmd := &cobra.Command{
		Use:   "delete [APP_NAME]",
		Short: "Delete an application from a project",
		Example: util.Doc(`
# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:

		export GIT_TOKEN=<token>
		export GIT_REPO=<repo_url>

# or with the flags:

		--git-token <token> --repo <repo_url>

# Get list of installed applications in a specifc project

	<BIN> app delete <app_name> --project <project_name>
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				log.G().Fatal("must enter application name")
			}

			appName = args[0]

			if opts.ProjectName == "" && !global {
				log.G().Fatal("must enter project name OR use '--global' flag")
			}

			return RunAppDelete(cmd.Context(), &AppDeleteOptions{
				BaseOptions: *opts,
				AppName:     appName,
				Global:      global,
			})
		},
	}

	cmd.Flags().BoolVarP(&global, "global", "g", false, "global")

	return cmd
}

func RunAppDelete(ctx context.Context, opts *AppDeleteOptions) error {
	r, repofs, err := prepareRepo(ctx, &opts.BaseOptions)
	if err != nil {
		return err
	}

	appDir := repofs.Join(store.Default.AppsDir, opts.AppName)
	appExists := repofs.ExistsOrDie(appDir)
	if !appExists {
		return fmt.Errorf(util.Doc(fmt.Sprintf("application '%s' not found", opts.AppName)))
	}

	var dirToRemove string
	commitMsg := fmt.Sprintf("Deleted app '%s'", opts.AppName)
	if opts.Global {
		dirToRemove = appDir
	} else {
		appOverlaysDir := repofs.Join(appDir, store.Default.OverlaysDir)
		projectDir := repofs.Join(appOverlaysDir, opts.ProjectName)
		overlayExists := repofs.ExistsOrDie(projectDir)
		if !overlayExists {
			return fmt.Errorf(util.Doc(fmt.Sprintf("application '%s' not found in project '%s'", opts.AppName, opts.ProjectName)))
		}

		allOverlays, err := repofs.ReadDir(appOverlaysDir)
		if err != nil {
			return fmt.Errorf("failed to read overlays directory '%s': %w", appOverlaysDir, err)
		}

		if len(allOverlays) == 1 {
			dirToRemove = appDir
		} else {
			commitMsg += fmt.Sprintf(" from project '%s'", opts.ProjectName)
			dirToRemove = projectDir
		}
	}

	err = billyUtils.RemoveAll(repofs, dirToRemove)
	if err != nil {
		return fmt.Errorf("failed to delete directory '%s': %w", dirToRemove, err)
	}

	log.G().Info("committing changes to gitops repo...")
	if err = r.Persist(ctx, &git.PushOptions{CommitMsg: commitMsg}); err != nil {
		return fmt.Errorf("failed to push to repo: %w", err)
	}

	return nil
}
