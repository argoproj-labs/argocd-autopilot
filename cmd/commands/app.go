package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"text/tabwriter"

	"github.com/argoproj/argocd-autopilot/pkg/application"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	"github.com/ghodss/yaml"
	memfs "github.com/go-git/go-billy/v5/memfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/spf13/cobra"
	kusttypes "sigs.k8s.io/kustomize/api/types"
)

var (
	ErrAppAlreadyInstalledOnProject = errors.New("application already installed on project")
	ErrAppCollisionWithExistingBase = errors.New("an application with the same name and a different base already exists, consider choosing a different name")
)

type (
	AppCreateOptions struct {
		ProjectName  string
		FS           fs.FS
		AppOpts      *application.CreateOptions
		CloneOptions *git.CloneOptions
	}
	AppListOptions struct {
		ProjectName  string
		FS           fs.FS
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
	cmd.AddCommand(NewAppListCommand())

	return cmd
}

func NewAppCreateCommand() *cobra.Command {
	var (
		projectName string
		appOpts     *application.CreateOptions
		cloneOpts   *git.CloneOptions
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
	
		--token <token> --repo <repo_url>
		
# Create a new application from kustomization in a remote repository
	
	<BIN> app create <new_app_name> --app github.com/some_org/some_repo/manifests?ref=v1.2.3 --project project_name
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				log.G().Fatal("must enter application name")
			}

			appOpts.AppName = args[0]

			return RunAppCreate(cmd.Context(), &AppCreateOptions{
				ProjectName:  projectName,
				FS:           fs.Create(memfs.New()),
				AppOpts:      appOpts,
				CloneOptions: cloneOpts,
			})
		},
	}
	appOpts = application.AddFlags(cmd)
	cloneOpts, err := git.AddFlags(cmd)
	die(err)

	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Project name")

	die(cmd.MarkFlagRequired("project"))
	die(cmd.MarkFlagRequired("app"))

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

	projectExists := opts.FS.ExistsOrDie(opts.FS.Join(store.Default.ProjectsDir, opts.ProjectName+".yaml"))
	if !projectExists {
		return fmt.Errorf(util.Doc("project '%[1]s' not found, please execute `<BIN> project create %[1]s`"), opts.ProjectName)
	}
	log.G().Debug("repository is ok")

	app, err := opts.AppOpts.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse application from flags: %v", err)
	}

	if err = createApplicationFiles(opts.FS, app, opts.ProjectName); err != nil {
		if errors.Is(err, ErrAppAlreadyInstalledOnProject) {
			return fmt.Errorf("application '%s' already exists in project '%s': %w", app.Name(), opts.ProjectName, ErrAppAlreadyInstalledOnProject)
		}

		return err
	}

	log.G().Info("committing changes to gitops repo...")
	if err = r.Persist(ctx, &git.PushOptions{CommitMsg: getCommitMsg(opts)}); err != nil {
		return fmt.Errorf("failed to push to repo: %w", err)
	}
	log.G().Infof("installed application: %s", opts.AppOpts.AppName)

	return nil
}

func createApplicationFiles(repoFS fs.FS, app application.Application, projectName string) error {
	basePath := repoFS.Join(store.Default.KustomizeDir, app.Name(), "base")
	overlayPath := repoFS.Join(store.Default.KustomizeDir, app.Name(), "overlays", projectName)

	// create Base
	baseKustomizationPath := repoFS.Join(basePath, "kustomization.yaml")
	baseKustomizationYAML, err := yaml.Marshal(app.Base())
	if err != nil {
		return fmt.Errorf("failed to marshal app base kustomization: %w", err)
	}

	if exists, err := writeApplicationFile(repoFS, baseKustomizationPath, "base", baseKustomizationYAML); err != nil {
		return err
	} else if exists {
		// check if the bases are the same
		log.G().Debug("application base with the same name exists, checking for collisions")
		if collision, err := checkBaseCollision(repoFS, baseKustomizationPath, app.Base()); err != nil {
			return err
		} else if collision {
			return ErrAppCollisionWithExistingBase
		}
	}

	// create Overlay
	overlayKustomizationPath := repoFS.Join(overlayPath, "kustomization.yaml")
	overlayKustomizationYAML, err := yaml.Marshal(app.Overlay())
	if err != nil {
		return fmt.Errorf("failed to marshal app overlay kustomization: %w", err)
	}
	if exists, err := writeApplicationFile(repoFS, overlayKustomizationPath, "overlay", overlayKustomizationYAML); err != nil {
		return err
	} else if exists {
		return ErrAppAlreadyInstalledOnProject
	}

	// get manifests - only used in flat installation mode
	if app.Manifests() != nil {
		manifestsPath := repoFS.Join(basePath, "install.yaml")
		if _, err = writeApplicationFile(repoFS, manifestsPath, "manifests", app.Manifests()); err != nil {
			return err
		}
	}

	// if we override the namespace we also need to write the namespace manifests next to the overlay
	if app.Namespace() != nil {
		nsPath := repoFS.Join(overlayPath, "namespace.yaml")
		nsYAML, err := yaml.Marshal(app.Namespace())
		if err != nil {
			return fmt.Errorf("failed to marshal app overlay namespace: %w", err)
		}

		if _, err = writeApplicationFile(repoFS, nsPath, "application namespace", nsYAML); err != nil {
			return err
		}
	}

	configPath := repoFS.Join(overlayPath, "config.json")
	config, err := json.Marshal(app.Config())
	if err != nil {
		return fmt.Errorf("failed to marshal app config.json: %w", err)
	}
	if _, err = writeApplicationFile(repoFS, configPath, "config", config); err != nil {
		return err
	}

	return nil
}

func checkBaseCollision(repoFS fs.FS, orgBasePath string, newBase *kusttypes.Kustomization) (bool, error) {
	f, err := repoFS.Open(orgBasePath)
	if err != nil {
		return false, err
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return false, err
	}

	orgBase := &kusttypes.Kustomization{}
	if err = yaml.Unmarshal(data, orgBase); err != nil {
		return false, err
	}

	return !reflect.DeepEqual(orgBase, newBase), nil
}

func writeApplicationFile(repoFS fs.FS, path, name string, data []byte) (bool, error) {
	absPath := repoFS.Join(repoFS.Root(), path)
	exists, err := repoFS.CheckExistsOrWrite(path, data)
	if err != nil {
		return false, fmt.Errorf("failed to create '%s' file at '%s': %w", name, absPath, err)
	} else if exists {
		log.G().Infof("'%s' file exists in '%s'", name, absPath)
		return true, nil
	}
	log.G().Infof("created '%s' file at '%s'", name, absPath)
	return false, nil
}

func getCommitMsg(opts *AppCreateOptions) string {
	commitMsg := fmt.Sprintf("installed app '%s' on project '%s'", opts.AppOpts.AppName, opts.ProjectName)
	if opts.CloneOptions.RepoRoot != "" {
		commitMsg += fmt.Sprintf(" installation-path: '%s'", opts.CloneOptions.RepoRoot)
	}
	return commitMsg
}
func NewAppListCommand() *cobra.Command {
	var (
		projectName string
		//	appListOpts   *application.
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
	
		--token <token> --repo <repo_url>
		
# Get list of installed applications in a specifc project
	
	<BIN> app list <project_name>
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				log.G().Fatal("must enter a project name")
			}
			projectName = args[0]

			return RunAppList(cmd.Context(), &AppListOptions{
				ProjectName:  projectName,
				FS:           fs.Create(memfs.New()),
				CloneOptions: cloneOpts,
			})
		},
	}
	cloneOpts, err := git.AddFlags(cmd)
	util.Die(err)

	return cmd
}

func RunAppList(ctx context.Context, opts *AppListOptions) error {

	var (
		err error
	)

	log.G().WithFields(log.Fields{
		"repoURL":  opts.CloneOptions.URL,
		"revision": opts.CloneOptions.Revision,
	}).Debug("starting with options: ")

	// clone repo
	log.G().Infof("cloning git repository: %s", opts.CloneOptions.URL)
	_, opts.FS, err = opts.CloneOptions.Clone(ctx, opts.FS)
	if err != nil {
		return err
	}

	log.G().Infof("using revision: \"%s\", installation path: \"%s\"", opts.CloneOptions.Revision, opts.FS.Root())
	if !opts.FS.ExistsOrDie(store.Default.BootsrtrapDir) {
		log.G().Fatalf("Bootstrap folder not found, please execute `repo bootstrap --installation-path %s` command", opts.FS.Root())
	}

	projExists := opts.FS.ExistsOrDie(opts.FS.Join(store.Default.ProjectsDir, opts.ProjectName+".yaml"))
	if !projExists {
		log.G().Fatalf(util.Doc(fmt.Sprintf("project '%[1]s' not found, please execute `<BIN> project create %[1]s`", opts.ProjectName)))
	}

	log.G().Debug("repository is ok")

	// get all apps beneath kustomize <project>\overlayes

	matches, err := billyUtils.Glob(opts.FS, fmt.Sprintf("/kustomize/*/overlays/%s", opts.ProjectName))
	if err != nil {
		log.G().Fatalf("failed to run glob on %s", opts.ProjectName)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "PROJECT\tNAME\tDEST_NAMESPACE\tDEST_SERVER\t\n")

	for _, appName := range matches {

		conf, err := getConfigFileFromPath(opts.FS, appName)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", opts.ProjectName, conf.UserGivenName, conf.DestNamespace, conf.DestServer)
	}
	_ = w.Flush()
	return nil

}
func getConfigFileFromPath(fs fs.FS, appName string) (*application.Config, error) {

	confFileName := fmt.Sprintf("%s/config.json", appName)
	file, err := fs.Open(confFileName)
	if err != nil {
		return nil, fmt.Errorf("%s not found", confFileName)
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s", confFileName)
	}
	conf := application.Config{}
	err = json.Unmarshal(b, &conf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %s", confFileName)
	}
	return &conf, nil

}
