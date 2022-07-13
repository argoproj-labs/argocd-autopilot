package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/argoproj-labs/argocd-autopilot/pkg/application"
	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git"
	"github.com/argoproj-labs/argocd-autopilot/pkg/kube"
	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/spf13/cobra"
)

type (
	AppCreateOptions struct {
		CloneOpts       *git.CloneOptions
		AppsCloneOpts   *git.CloneOptions
		ProjectName     string
		KubeContextName string
		AppOpts         *application.CreateOptions
		KubeFactory     kube.Factory
		Timeout         time.Duration
		Labels          map[string]string
		Include         string
		Exclude         string
	}

	AppDeleteOptions struct {
		CloneOpts   *git.CloneOptions
		ProjectName string
		AppName     string
		Global      bool
	}

	AppListOptions struct {
		CloneOpts   *git.CloneOptions
		ProjectName string
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

	cmd.AddCommand(NewAppCreateCommand())
	cmd.AddCommand(NewAppListCommand())
	cmd.AddCommand(NewAppDeleteCommand())

	return cmd
}

func NewAppCreateCommand() *cobra.Command {
	var (
		cloneOpts     *git.CloneOptions
		appsCloneOpts *git.CloneOptions
		appOpts       *application.CreateOptions
		projectName   string
		timeout       time.Duration
		f             kube.Factory
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

# using the --type flag (kustomize|dir) is optional. If it is ommitted, <BIN> will clone
# the --app repository, and infer the type automatically.

# Create a new application from kustomization in a remote repository (will reference the HEAD revision)

	<BIN> app create <new_app_name> --app github.com/some_org/some_repo/manifests --project project_name

# Reference a specific git commit hash:

  <BIN> app create <new_app_name> --app github.com/some_org/some_repo/manifests?sha=<commit_hash> --project project_name

# Reference a specific git tag:

  <BIN> app create <new_app_name> --app github.com/some_org/some_repo/manifests?tag=<tag_name> --project project_name

# Reference a specific git branch:

  <BIN> app create <new_app_name> --app github.com/some_org/some_repo/manifests?ref=<branch_name> --project project_name

# Wait until the application is Synced in the cluster:

  <BIN> app create <new_app_name> --app github.com/some_org/some_repo/manifests --project project_name --wait-timeout 2m --context my_context 
`),
		PreRun: func(_ *cobra.Command, _ []string) {
			cloneOpts.Parse()
			appsCloneOpts.Parse()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if len(args) < 1 {
				log.G(ctx).Fatal("must enter application name")
			}

			appOpts.AppName = args[0]
			return RunAppCreate(ctx, &AppCreateOptions{
				CloneOpts:       cloneOpts,
				AppsCloneOpts:   appsCloneOpts,
				ProjectName:     projectName,
				KubeContextName: cmd.Flag("context").Value.String(),
				AppOpts:         appOpts,
				Timeout:         timeout,
				KubeFactory:     f,
			})
		},
	}

	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Project name")
	cmd.Flags().DurationVar(&timeout, "wait-timeout", time.Duration(0), "If not '0s', will try to connect to the cluster and wait until the application is in 'Synced' status for the specified timeout period")
	cloneOpts = git.AddFlags(cmd, &git.AddFlagsOptions{
		FS:            memfs.New(),
		CloneForWrite: true,
	})
	appsCloneOpts = git.AddFlags(cmd, &git.AddFlagsOptions{
		FS:       memfs.New(),
		Prefix:   "apps",
		Optional: true,
	})
	appOpts = application.AddFlags(cmd)
	f = kube.AddFlags(cmd.Flags())

	die(cmd.MarkFlagRequired("app"))
	die(cmd.MarkFlagRequired("project"))

	return cmd
}

func RunAppCreate(ctx context.Context, opts *AppCreateOptions) error {
	var (
		appsRepo git.Repository
		appsfs   fs.FS
	)

	log.G(ctx).WithFields(log.Fields{
		"app-url":      opts.AppsCloneOpts.URL(),
		"app-revision": opts.AppsCloneOpts.Revision(),
		"app-path":     opts.AppsCloneOpts.Path(),
	}).Debug("starting with options: ")

	r, repofs, err := prepareRepo(ctx, opts.CloneOpts, opts.ProjectName)
	if err != nil {
		return err
	}

	if opts.AppsCloneOpts.Repo != "" {
		if opts.AppsCloneOpts.Auth.Password == "" {
			opts.AppsCloneOpts.Auth.Username = opts.CloneOpts.Auth.Username
			opts.AppsCloneOpts.Auth.Password = opts.CloneOpts.Auth.Password
		}

		appsRepo, appsfs, err = getRepo(ctx, opts.AppsCloneOpts)
		if err != nil {
			return err
		}
	} else {
		opts.AppsCloneOpts = opts.CloneOpts
		appsRepo, appsfs = r, repofs
	}

	if err = setAppOptsDefaults(ctx, repofs, opts); err != nil {
		return err
	}

	app, err := parseApp(opts.AppOpts, opts.ProjectName, opts.CloneOpts.URL(), opts.CloneOpts.Revision(), opts.CloneOpts.Path())
	if err != nil {
		return fmt.Errorf("failed to parse application from flags: %w", err)
	}

	if err = app.CreateFiles(repofs, appsfs, opts.ProjectName); err != nil {
		if errors.Is(err, application.ErrAppAlreadyInstalledOnProject) {
			return fmt.Errorf("application '%s' already exists in project '%s': %w", app.Name(), opts.ProjectName, err)
		}

		return err
	}

	if opts.AppsCloneOpts != opts.CloneOpts {
		log.G(ctx).Info("committing changes to apps repo...")
		if _, err = appsRepo.Persist(ctx, &git.PushOptions{CommitMsg: getCommitMsg(opts, appsfs)}); err != nil {
			return fmt.Errorf("failed to push to apps repo: %w", err)
		}
	}

	log.G(ctx).Info("committing changes to gitops repo...")
	revision, err := r.Persist(ctx, &git.PushOptions{CommitMsg: getCommitMsg(opts, repofs)})
	if err != nil {
		return fmt.Errorf("failed to push to gitops repo: %w", err)
	}

	if opts.Timeout > 0 {
		namespace, err := getInstallationNamespace(repofs)
		if err != nil {
			return fmt.Errorf("failed to get application namespace: %w", err)
		}

		log.G(ctx).WithField("timeout", opts.Timeout).Infof("waiting for '%s' to finish syncing", opts.AppOpts.AppName)
		fullName := fmt.Sprintf("%s-%s", opts.ProjectName, opts.AppOpts.AppName)

		// wait for argocd to be ready before applying argocd-apps
		stop := util.WithSpinner(ctx, fmt.Sprintf("waiting for '%s' to be ready", fullName))
		if err = waitAppSynced(ctx, opts.KubeFactory, opts.Timeout, fullName, namespace, revision, true); err != nil {
			stop()
			return fmt.Errorf("failed waiting for application to sync: %w", err)
		}

		stop()
	}

	log.G(ctx).Infof("installed application: %s", opts.AppOpts.AppName)
	return nil
}

var setAppOptsDefaults = func(ctx context.Context, repofs fs.FS, opts *AppCreateOptions) error {
	var err error

	if opts.AppOpts.DestServer == store.Default.DestServer || opts.AppOpts.DestServer == "" {
		opts.AppOpts.DestServer, err = getProjectDestServer(repofs, opts.ProjectName)
		if err != nil {
			return err
		}
	}

	if opts.AppOpts.DestNamespace == "" {
		opts.AppOpts.DestNamespace = "default"
	}

	if opts.AppOpts.Labels == nil {
		opts.AppOpts.Labels = opts.Labels
	}

	if opts.AppOpts.AppType != "" {
		return nil
	}

	var fsys fs.FS
	if _, err := os.Stat(opts.AppOpts.AppSpecifier); err == nil {
		// local directory
		fsys = fs.Create(osfs.New(opts.AppOpts.AppSpecifier))
	} else {
		host, orgRepo, p, _, _, suffix, _ := util.ParseGitUrl(opts.AppOpts.AppSpecifier)
		url := host + orgRepo + suffix
		log.G(ctx).Infof("cloning repo: '%s', to infer app type from path '%s'", url, p)
		cloneOpts := &git.CloneOptions{
			Repo: opts.AppOpts.AppSpecifier,
			Auth: opts.CloneOpts.Auth,
			FS:   fs.Create(memfs.New()),
		}
		cloneOpts.Parse()
		_, fsys, err = getRepo(ctx, cloneOpts)
		if err != nil {
			return err
		}
	}

	opts.AppOpts.AppType = application.InferAppType(fsys)
	log.G(ctx).Infof("inferred application type: %s", opts.AppOpts.AppType)

	return nil
}

var parseApp = func(appOpts *application.CreateOptions, projectName, repoURL, targetRevision, repoRoot string) (application.Application, error) {
	return appOpts.Parse(projectName, repoURL, targetRevision, repoRoot)
}

func getProjectDestServer(repofs fs.FS, projectName string) (string, error) {
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

func NewAppListCommand() *cobra.Command {
	var (
		cloneOpts *git.CloneOptions
	)

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
		PreRun: func(_ *cobra.Command, _ []string) { cloneOpts.Parse() },
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if len(args) < 1 {
				log.G(ctx).Fatal("must enter a project name")
			}

			return RunAppList(ctx, &AppListOptions{
				CloneOpts:   cloneOpts,
				ProjectName: args[0],
			})
		},
	}

	cloneOpts = git.AddFlags(cmd, &git.AddFlagsOptions{
		FS: memfs.New(),
	})

	return cmd
}

func RunAppList(ctx context.Context, opts *AppListOptions) error {
	_, repofs, err := prepareRepo(ctx, opts.CloneOpts, opts.ProjectName)
	if err != nil {
		return err
	}

	// get all apps beneath apps/*/overlays/<project>
	matches, err := billyUtils.Glob(repofs, repofs.Join(store.Default.AppsDir, "*", store.Default.OverlaysDir, opts.ProjectName))
	if err != nil {
		log.G(ctx).Fatalf("failed to run glob on %s", opts.ProjectName)
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

func NewAppDeleteCommand() *cobra.Command {
	var (
		cloneOpts   *git.CloneOptions
		projectName string
		global      bool
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
		PreRun: func(_ *cobra.Command, _ []string) { cloneOpts.Parse() },
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if len(args) < 1 {
				log.G(ctx).Fatal("must enter application name")
			}

			if projectName == "" && !global {
				log.G(ctx).Fatal("must enter project name OR use '--global' flag")
			}

			return RunAppDelete(ctx, &AppDeleteOptions{
				CloneOpts:   cloneOpts,
				ProjectName: projectName,
				AppName:     args[0],
				Global:      global,
			})
		},
	}

	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Project name")
	cmd.Flags().BoolVarP(&global, "global", "g", false, "global")

	cloneOpts = git.AddFlags(cmd, &git.AddFlagsOptions{
		FS:            memfs.New(),
		CloneForWrite: true,
	})

	return cmd
}

func RunAppDelete(ctx context.Context, opts *AppDeleteOptions) error {
	r, repofs, err := prepareRepo(ctx, opts.CloneOpts, opts.ProjectName)
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
		overlaysExists := repofs.ExistsOrDie(appOverlaysDir)
		if !overlaysExists {
			appOverlaysDir = appDir
		}

		appProjectDir := repofs.Join(appOverlaysDir, opts.ProjectName)
		overlayExists := repofs.ExistsOrDie(appProjectDir)
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
			dirToRemove = appProjectDir
		}
	}

	err = billyUtils.RemoveAll(repofs, dirToRemove)
	if err != nil {
		return fmt.Errorf("failed to delete directory '%s': %w", dirToRemove, err)
	}

	log.G(ctx).Info("committing changes to gitops repo...")
	if _, err = r.Persist(ctx, &git.PushOptions{CommitMsg: commitMsg}); err != nil {
		return fmt.Errorf("failed to push to repo: %w", err)
	}

	return nil
}
