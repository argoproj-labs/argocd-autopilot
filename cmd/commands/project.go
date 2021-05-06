package commands

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/argoproj/argocd-autopilot/pkg/argocd"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	appset "github.com/argoproj-labs/applicationset/api/v1alpha1"
	appsetv1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/ghodss/yaml"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var DefaultApplicationSetGeneratorInterval int64 = 20

type (
	ProjectCreateOptions struct {
		Name            string
		DestKubeContext string
		DryRun          bool
		FS              fs.FS
		CloneOptions    *git.CloneOptions
		AddCmd          argocd.AddClusterCmd
	}
)

func NewProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project",
		Aliases: []string{"proj"},
		Short:   "Manage projects",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
			exit(1)
		},
	}

	opts, err := addFlags(cmd)
	die(err)
	cmd.AddCommand(NewProjectCreateCommand())
	cmd.AddCommand(NewProjectListCommand(opts))

	return cmd
}

func NewProjectCreateCommand() *cobra.Command {
	var (
		name        string
		kubeContext string
		dryRun      bool
		addCmd      argocd.AddClusterCmd
		cloneOpts   *git.CloneOptions
	)

	cmd := &cobra.Command{
		Use:   "create [PROJECT]",
		Short: "Create a new project",
		Example: util.Doc(`
# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:

		export GIT_TOKEN=<token>
		export GIT_REPO=<repo_url>

# or with the flags:

		--git-token <token> --repo <repo_url>

# Create a new project

	<BIN> project create <PROJECT_NAME>

# Create a new project in a specific path inside the GitOps repo

  <BIN> project create <PROJECT_NAME> --installation-path path/to/installation_root
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				log.G().Fatal("must enter project name")
			}
			name = args[0]

			return RunProjectCreate(cmd.Context(), &ProjectCreateOptions{
				Name:            name,
				DestKubeContext: kubeContext,
				DryRun:          dryRun,
				FS:              fs.Create(memfs.New()),
				CloneOptions:    cloneOpts,
				AddCmd:          addCmd,
			})
		},
	}

	cmd.Flags().StringVar(&kubeContext, "dest-kube-context", "", "The default destination kubernetes context for applications in this project")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "If true, print manifests instead of applying them to the cluster (nothing will be commited to git)")

	addCmd, err := argocd.AddClusterAddFlags(cmd)
	die(err)

	cloneOpts, err = git.AddFlags(cmd)
	die(err)

	return cmd
}

func RunProjectCreate(ctx context.Context, opts *ProjectCreateOptions) error {
	var (
		err                   error
		installationNamespace string
		r                     git.Repository
	)

	log.G().WithFields(log.Fields{
		"prject":       opts.Name,
		"repoURL":      opts.CloneOptions.URL,
		"revision":     opts.CloneOptions.Revision,
		"installation": opts.CloneOptions.RepoRoot,
	}).Debug("starting with options: ")

	log.G().Infof("cloning repo: %s", opts.CloneOptions.URL)

	// clone GitOps repo
	r, opts.FS, err = clone(ctx, opts.CloneOptions, opts.FS)
	if err != nil {
		return err
	}

	log.G().Infof("using revision: \"%s\", installation path: \"%s\"", opts.CloneOptions.Revision, opts.FS.Root())

	installationNamespace, err = getInstallationNamespace(opts.FS)
	if err != nil {
		return fmt.Errorf(util.Doc("Bootstrap folder not found, please execute `<BIN> repo bootstrap --installation-path %s` command"), opts.FS.Root())
	}

	projectExists := opts.FS.ExistsOrDie(opts.FS.Join(store.Default.ProjectsDir, opts.Name+".yaml"))
	if projectExists {
		return fmt.Errorf("project '%s' already exists", opts.Name)
	}
	log.G().Debug("repository is ok")

	destServer := store.Default.DestServer
	if opts.DestKubeContext != "" {
		destServer, err = util.KubeContextToServer(opts.DestKubeContext)
		if err != nil {
			return err
		}
	}

	project, appSet := generateProject(&GenerateProjectOptions{
		Name:              opts.Name,
		Namespace:         installationNamespace,
		RepoURL:           opts.CloneOptions.URL,
		Revision:          opts.CloneOptions.Revision,
		InstallationPath:  opts.CloneOptions.RepoRoot,
		DefaultDestServer: destServer,
	})

	projectYAML, err := yaml.Marshal(project)
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	appsetYAML, err := yaml.Marshal(appSet)
	if err != nil {
		return fmt.Errorf("failed to marshal appSet: %w", err)
	}

	joinedYAML := util.JoinManifests(projectYAML, appsetYAML)

	if opts.DryRun {
		log.G().Printf("%s", joinedYAML)
		return nil
	}

	if opts.DestKubeContext != "" {
		log.G().Infof("adding cluster: %s", opts.DestKubeContext)
		if err = opts.AddCmd.Execute(ctx, opts.DestKubeContext); err != nil {
			return fmt.Errorf("failed to add new cluster credentials: %w", err)
		}
	}

	if _, err = opts.FS.WriteFile(opts.FS.Join(store.Default.ProjectsDir, opts.Name+".yaml"), joinedYAML); err != nil {
		return fmt.Errorf("failed to create project file: %w", err)
	}

	log.G().Infof("pushing new project manifest to repo")
	if err = r.Persist(ctx, &git.PushOptions{CommitMsg: "Added project " + opts.Name}); err != nil {
		return err
	}
	log.G().Infof("project created: '%s'", opts.Name)

	return nil
}

type GenerateProjectOptions struct {
	Name              string
	Namespace         string
	DefaultDestServer string
	RepoURL           string
	Revision          string
	InstallationPath  string
}

var generateProject = func(o *GenerateProjectOptions) (*argocdv1alpha1.AppProject, *appset.ApplicationSet) {
	appProject := &argocdv1alpha1.AppProject{
		TypeMeta: metav1.TypeMeta{
			Kind:       argocdv1alpha1.AppProjectSchemaGroupVersionKind.Kind,
			APIVersion: argocdv1alpha1.AppProjectSchemaGroupVersionKind.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.Name,
			Namespace: o.Namespace,
			Annotations: map[string]string{
				"argocd.argoproj.io/sync-options":  "PruneLast=true",
				store.Default.DestServerAnnotation: o.DefaultDestServer,
			},
		},
		Spec: argocdv1alpha1.AppProjectSpec{
			SourceRepos: []string{"*"},
			Destinations: []argocdv1alpha1.ApplicationDestination{
				{
					Server:    "*",
					Namespace: "*",
				},
			},
			Description: fmt.Sprintf("%s project", o.Name),
			ClusterResourceWhitelist: []metav1.GroupKind{
				{
					Group: "*",
					Kind:  "*",
				},
			},
			NamespaceResourceWhitelist: []metav1.GroupKind{
				{
					Group: "*",
					Kind:  "*",
				},
			},
		},
	}

	appSet := &appset.ApplicationSet{
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
						RequeueAfterSeconds: &DefaultApplicationSetGeneratorInterval,
					},
				},
			},
			Template: appset.ApplicationSetTemplate{
				ApplicationSetTemplateMeta: appset.ApplicationSetTemplateMeta{
					Namespace: o.Namespace,
					Name:      fmt.Sprintf("%s-{{userGivenName}}", o.Name),
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": store.Default.ManagedBy,
						"app.kubernetes.io/name":       "{{appName}}",
					},
				},
				Spec: appsetv1alpha1.ApplicationSpec{
					Project: o.Name,
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

	return appProject, appSet
}

var getInstallationNamespace = func(repofs fs.FS) (string, error) {
	f, err := repofs.Open(repofs.Join(store.Default.BootsrtrapDir, store.Default.ArgoCDName+".yaml"))
	if err != nil {
		return "", err
	}

	d, err := ioutil.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("failed to read namespace file: %w", err)
	}

	a := &appsetv1alpha1.Application{}
	if err = yaml.Unmarshal(d, a); err != nil {
		return "", fmt.Errorf("failed to unmarshal namespace: %w", err)
	}

	return a.Spec.Destination.Namespace, nil
}

type (
	ProjectListOptions struct {
		BaseOptions
		Out io.Writer
	}
)

func NewProjectListCommand(opts *BaseOptions) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "list ",
		Short: "Lists all the projects on a git repository",
		Example: util.Doc(`
# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:

		export GIT_TOKEN=<token>
		export GIT_REPO=<repo_url>

# or with the flags:

		--git-token <token> --repo <repo_url>

# Lists projects

	<BIN> project list
`),
		RunE: func(cmd *cobra.Command, args []string) error {

			return RunProjectList(cmd.Context(), &ProjectListOptions{
				BaseOptions: *opts,
				Out:         os.Stdout,
			})
		},
	}

	return cmd
}

func RunProjectList(ctx context.Context, opts *ProjectListOptions) error {

	_, repofs, err := prepareRepo(ctx, &opts.BaseOptions)
	if err != nil {
		return err
	}

	matches, err := glob(repofs, repofs.Join(store.Default.ProjectsDir, "*.yaml"))
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(opts.Out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tNAMESPACE\tCLUSTER\t\n")

	for _, name := range matches {
		proj, _, err := getProjectInfoFromFile(repofs, name)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", proj.Name, proj.Namespace, proj.ClusterName)

	}
	w.Flush()
	return nil
}

var getProjectInfoFromFile = func(fs fs.FS, name string) (*argocdv1alpha1.AppProject, *appsetv1alpha1.ApplicationSpec, error) {
	file, err := fs.Open(name)
	if err != nil {
		return nil, nil, fmt.Errorf("%s not found", name)
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file %s", name)
	}
	yamls := util.SplitManifests(b)
	if len(yamls) != 2 {
		return nil, nil, fmt.Errorf("expected 2 files when splitting %s", name)
	}
	proj := argocdv1alpha1.AppProject{}
	err = yaml.Unmarshal(yamls[0], &proj)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal %s", name)
	}
	appSet := appsetv1alpha1.ApplicationSpec{}
	err = yaml.Unmarshal(yamls[1], &proj)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal %s", name)
	}

	return &proj, &appSet, nil

}
