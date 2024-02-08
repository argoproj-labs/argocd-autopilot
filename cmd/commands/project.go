package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"text/tabwriter"

	"github.com/argoproj-labs/argocd-autopilot/pkg/application"
	"github.com/argoproj-labs/argocd-autopilot/pkg/argocd"
	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	fsutils "github.com/argoproj-labs/argocd-autopilot/pkg/fs/utils"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git"
	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/go-git/go-billy/v5/memfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	ProjectCreateOptions struct {
		CloneOpts       *git.CloneOptions
		ProjectName     string
		DestKubeServer  string
		DestKubeContext string
		DryRun          bool
		AddCmd          argocd.AddClusterCmd
		Labels          map[string]string
		Annotations     map[string]string
	}

	ProjectDeleteOptions struct {
		CloneOpts   *git.CloneOptions
		ProjectName string
	}

	ProjectListOptions struct {
		CloneOpts *git.CloneOptions
		Out       io.Writer
	}

	GenerateProjectOptions struct {
		Name               string
		Namespace          string
		DefaultDestServer  string
		DefaultDestContext string
		RepoURL            string
		Revision           string
		InstallationPath   string
		Labels             map[string]string
		Annotations        map[string]string
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

	cmd.AddCommand(NewProjectCreateCommand())
	cmd.AddCommand(NewProjectListCommand())
	cmd.AddCommand(NewProjectDeleteCommand())

	return cmd
}

func NewProjectCreateCommand() *cobra.Command {
	var (
		kubeServer  string
		kubeContext string
		dryRun      bool
		addCmd      argocd.AddClusterCmd
		labels      map[string]string
		annotations map[string]string
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
`),
		PreRun: func(_ *cobra.Command, _ []string) { cloneOpts.Parse() },
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if len(args) < 1 {
				log.G(ctx).Fatal("must enter project name")
			}

			return RunProjectCreate(ctx, &ProjectCreateOptions{
				CloneOpts:       cloneOpts,
				ProjectName:     args[0],
				DestKubeServer:  kubeServer,
				DestKubeContext: kubeContext,
				DryRun:          dryRun,
				AddCmd:          addCmd,
				Labels:          labels,
				Annotations:     annotations,
			})
		},
	}

	cmd.Flags().StringVar(&kubeServer, "dest-server", "", "The default destination kubernetes server for applications in this project")
	cmd.Flags().StringVar(&kubeContext, "dest-kube-context", "", "The default destination kubernetes context for applications in this project (will be ignored if --dest-server is supplied)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "If true, print manifests instead of applying them to the cluster (nothing will be commited to git)")
	cmd.Flags().StringToStringVar(&labels, "labels", nil, "Optional labels that will be set on the Application resource. (e.g. \"app.kubernetes.io/managed-by={{ placeholder }}\"")
	cmd.Flags().StringToStringVar(&annotations, "annotations", nil, "Optional annotations that will be set on the Application resource. (e.g. \"argocd.argoproj.io/sync-wave={{ placeholder }}\"")

	cloneOpts = git.AddFlags(cmd, &git.AddFlagsOptions{
		FS:            memfs.New(),
		CloneForWrite: true,
	})
	addCmd, err := argocd.AddClusterAddFlags(cmd)
	die(err)

	return cmd
}

func RunProjectCreate(ctx context.Context, opts *ProjectCreateOptions) error {
	var (
		err                   error
		installationNamespace string
	)

	r, repofs, err := prepareRepo(ctx, opts.CloneOpts, "")
	if err != nil {
		return err
	}

	installationNamespace, err = getInstallationNamespace(repofs)
	if err != nil {
		return fmt.Errorf(util.Doc("Bootstrap folder not found, please execute `<BIN> repo bootstrap --installation-path %s` command"), repofs.Root())
	}

	projectExists := repofs.ExistsOrDie(repofs.Join(store.Default.ProjectsDir, opts.ProjectName+".yaml"))
	if projectExists {
		return fmt.Errorf("project '%s' already exists", opts.ProjectName)
	}

	log.G(ctx).Debug("repository is ok")

	if opts.DestKubeServer == "" {
		opts.DestKubeServer = store.Default.DestServer
		if opts.DestKubeContext != "" {
			opts.DestKubeServer, err = util.KubeContextToServer(opts.DestKubeContext)
			if err != nil {
				return err
			}
		}
	}

	projectYAML, appsetYAML, clusterResReadme, clusterResConf, err := generateProjectManifests(&GenerateProjectOptions{
		Name:               opts.ProjectName,
		Namespace:          installationNamespace,
		RepoURL:            opts.CloneOpts.URL(),
		Revision:           opts.CloneOpts.Revision(),
		InstallationPath:   opts.CloneOpts.Path(),
		DefaultDestServer:  opts.DestKubeServer,
		DefaultDestContext: opts.DestKubeContext,
		Labels:             opts.Labels,
		Annotations:        opts.Annotations,
	})
	if err != nil {
		return fmt.Errorf("failed to generate project resources: %w", err)
	}

	if opts.DryRun {
		log.G(ctx).Printf("%s", util.JoinManifests(projectYAML, appsetYAML))
		return nil
	}

	bulkWrites := []fsutils.BulkWriteRequest{}

	if opts.DestKubeContext != "" {
		log.G(ctx).Infof("adding cluster: %s", opts.DestKubeContext)
		if err = opts.AddCmd.Execute(ctx, opts.DestKubeContext); err != nil {
			return fmt.Errorf("failed to add new cluster credentials: %w", err)
		}

		if !repofs.ExistsOrDie(repofs.Join(store.Default.BootsrtrapDir, store.Default.ClusterResourcesDir, opts.DestKubeContext)) {
			bulkWrites = append(bulkWrites, fsutils.BulkWriteRequest{
				Filename: repofs.Join(store.Default.BootsrtrapDir, store.Default.ClusterResourcesDir, opts.DestKubeContext+".json"),
				Data:     clusterResConf,
				ErrMsg:   "failed to write cluster config",
			})

			bulkWrites = append(bulkWrites, fsutils.BulkWriteRequest{
				Filename: repofs.Join(store.Default.BootsrtrapDir, store.Default.ClusterResourcesDir, opts.DestKubeContext, "README.md"),
				Data:     clusterResReadme,
				ErrMsg:   "failed to write cluster resources readme",
			})
		}
	}

	bulkWrites = append(bulkWrites, fsutils.BulkWriteRequest{
		Filename: repofs.Join(store.Default.ProjectsDir, opts.ProjectName+".yaml"),
		Data:     util.JoinManifests(projectYAML, appsetYAML),
		ErrMsg:   "failed to create project file",
	})

	if err = fsutils.BulkWrite(repofs, bulkWrites...); err != nil {
		return err
	}

	log.G(ctx).Infof("pushing new project manifest to repo")
	if _, err = r.Persist(ctx, &git.PushOptions{CommitMsg: fmt.Sprintf("Added project '%s'", opts.ProjectName)}); err != nil {
		return err
	}

	log.G(ctx).Infof("project created: '%s'", opts.ProjectName)

	return nil
}

func generateProjectManifests(o *GenerateProjectOptions) (projectYAML, appSetYAML, clusterResReadme, clusterResConfig []byte, err error) {
	project := &argocdv1alpha1.AppProject{
		TypeMeta: metav1.TypeMeta{
			Kind:       argocdv1alpha1.AppProjectSchemaGroupVersionKind.Kind,
			APIVersion: argocdv1alpha1.AppProjectSchemaGroupVersionKind.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.Name,
			Namespace: o.Namespace,
			Annotations: map[string]string{
				"argocd.argoproj.io/sync-wave":     "-2",
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
	if projectYAML, err = yaml.Marshal(project); err != nil {
		err = fmt.Errorf("failed to marshal AppProject: %w", err)
		return
	}

	appSetYAML, err = createAppSet(&createAppSetOptions{
		name:                        o.Name,
		namespace:                   o.Namespace,
		appName:                     fmt.Sprintf("%s-{{ userGivenName }}", o.Name),
		appNamespace:                o.Namespace,
		appProject:                  o.Name,
		repoURL:                     "{{ srcRepoURL }}",
		srcPath:                     "{{ srcPath }}",
		revision:                    "{{ srcTargetRevision }}",
		destServer:                  "{{ destServer }}",
		destNamespace:               "{{ destNamespace }}",
		prune:                       true,
		preserveResourcesOnDeletion: false,
		appLabels:                   getDefaultAppLabels(o.Labels),
		appAnnotations:              o.Annotations,
		generators: []argocdv1alpha1.ApplicationSetGenerator{
			{
				Git: &argocdv1alpha1.GitGenerator{
					RepoURL:  o.RepoURL,
					Revision: o.Revision,
					Files: []argocdv1alpha1.GitFileGeneratorItem{
						{
							Path: path.Join(o.InstallationPath, store.Default.AppsDir, "**", o.Name, "config.json"),
						},
					},
					RequeueAfterSeconds: &DefaultApplicationSetGeneratorInterval,
				},
			},
			{
				Git: &argocdv1alpha1.GitGenerator{
					RepoURL:  o.RepoURL,
					Revision: o.Revision,
					Files: []argocdv1alpha1.GitFileGeneratorItem{
						{
							Path: path.Join(o.InstallationPath, store.Default.AppsDir, "**", o.Name, "config_dir.json"),
						},
					},
					RequeueAfterSeconds: &DefaultApplicationSetGeneratorInterval,
					Template: argocdv1alpha1.ApplicationSetTemplate{
						Spec: argocdv1alpha1.ApplicationSpec{
							Source: &argocdv1alpha1.ApplicationSource{
								Directory: &argocdv1alpha1.ApplicationSourceDirectory{
									Recurse: true,
									Exclude: "{{ exclude }}",
									Include: "{{ include }}",
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to marshal ApplicationSet: %w", err)
		return
	}

	clusterResReadme = []byte(strings.ReplaceAll(string(clusterResReadmeTpl), "{CLUSTER}", o.DefaultDestServer))

	clusterResConfig, err = json.Marshal(&application.ClusterResConfig{Name: o.DefaultDestContext, Server: o.DefaultDestServer})
	if err != nil {
		err = fmt.Errorf("failed to create cluster resources config: %w", err)
		return
	}

	return
}

func getDefaultAppLabels(labels map[string]string) map[string]string {
	res := map[string]string{
		store.Default.LabelKeyAppManagedBy: store.Default.LabelValueManagedBy,
		store.Default.LabelKeyAppName:      "{{ appName }}",
	}
	for k, v := range labels {
		res[k] = v
	}

	return res
}

func NewProjectListCommand() *cobra.Command {
	var (
		cloneOpts *git.CloneOptions
	)

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
		PreRun: func(_ *cobra.Command, _ []string) { cloneOpts.Parse() },
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunProjectList(cmd.Context(), &ProjectListOptions{
				CloneOpts: cloneOpts,
				Out:       os.Stdout,
			})
		},
	}

	cloneOpts = git.AddFlags(cmd, &git.AddFlagsOptions{
		FS: memfs.New(),
	})

	return cmd
}

func RunProjectList(ctx context.Context, opts *ProjectListOptions) error {
	_, repofs, err := prepareRepo(ctx, opts.CloneOpts, "")
	if err != nil {
		return err
	}

	matches, err := billyUtils.Glob(repofs, repofs.Join(store.Default.ProjectsDir, "*.yaml"))
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(opts.Out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\tNAMESPACE\tDEFAULT CLUSTER\t\n")

	for _, name := range matches {
		proj, _, err := getProjectInfoFromFile(repofs, name)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", proj.Name, proj.Namespace, proj.Annotations[store.Default.DestServerAnnotation])
	}

	w.Flush()
	return nil
}

var getProjectInfoFromFile = func(repofs fs.FS, name string) (*argocdv1alpha1.AppProject, *argocdv1alpha1.ApplicationSet, error) {
	proj := &argocdv1alpha1.AppProject{}
	appSet := &argocdv1alpha1.ApplicationSet{}
	if err := repofs.ReadYamls(name, proj, appSet); err != nil {
		return nil, nil, err
	}

	return proj, appSet, nil
}

func NewProjectDeleteCommand() *cobra.Command {
	var (
		cloneOpts *git.CloneOptions
	)

	cmd := &cobra.Command{
		Use:   "delete [PROJECT_NAME]",
		Short: "Delete a project and all of its applications",
		Example: util.Doc(`
# To run this command you need to create a personal access token for your git provider,
# and have a bootstrapped GitOps repository, and provide them using:
	
		export GIT_TOKEN=<token>
		export GIT_REPO=<repo_url>

# or with the flags:
	
		--token <token> --repo <repo_url>
		
# Delete a project
	
	<BIN> project delete <project_name>
`),
		PreRun: func(_ *cobra.Command, _ []string) { cloneOpts.Parse() },
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if len(args) < 1 {
				log.G(ctx).Fatal("must enter project name")
			}

			return RunProjectDelete(ctx, &ProjectDeleteOptions{
				CloneOpts:   cloneOpts,
				ProjectName: args[0],
			})
		},
	}

	cloneOpts = git.AddFlags(cmd, &git.AddFlagsOptions{
		FS:            memfs.New(),
		CloneForWrite: true,
	})

	return cmd
}

func RunProjectDelete(ctx context.Context, opts *ProjectDeleteOptions) error {
	r, repofs, err := prepareRepo(ctx, opts.CloneOpts, opts.ProjectName)
	if err != nil {
		return err
	}

	allApps, err := repofs.ReadDir(store.Default.AppsDir)
	if err != nil {
		return fmt.Errorf("failed to list all applications")
	}

	for _, app := range allApps {
		err = application.DeleteFromProject(repofs, app.Name(), opts.ProjectName)
		if err != nil {
			return err
		}
	}

	err = repofs.Remove(repofs.Join(store.Default.ProjectsDir, opts.ProjectName+".yaml"))
	if err != nil {
		return fmt.Errorf("failed to delete project '%s': %w", opts.ProjectName, err)
	}

	log.G(ctx).Info("committing changes to gitops repo...")
	if _, err = r.Persist(ctx, &git.PushOptions{CommitMsg: fmt.Sprintf("Deleted project '%s'", opts.ProjectName)}); err != nil {
		return fmt.Errorf("failed to push to repo: %w", err)
	}

	return nil
}
