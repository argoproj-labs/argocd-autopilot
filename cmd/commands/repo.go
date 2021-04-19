package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/argoproj/argocd-autopilot/pkg/application"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/kube"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	"github.com/ghodss/yaml"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	installationModeFlat   = "flat"
	installationModeNormal = "normal"
)

var supportedProviders = []string{"github"}

type (
	RepoCreateOptions struct {
		Provider string
		Owner    string
		Repo     string
		Token    string
		Public   bool
		Host     string
	}

	RepoBootstrapOptions struct {
		RepoURL          string
		Revision         string
		GitToken         string
		InstallationPath string
		InstallationMode string
		Namespace        string
		KubeContext      string
		Namespaced       bool
		DryRun           bool
		HidePassword     bool
		Timeout          time.Duration
		FS               fs.FS
		KubeFactory      kube.Factory
		AppOptions       *application.CreateOptions
		CloneOptions     *git.CloneOptions
	}
)

func NewRepoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage gitops repositories",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
			os.Exit(1)
		},
	}

	cmd.AddCommand(NewRepoCreateCommand())
	cmd.AddCommand(NewRepoBootstrapCommand())

	return cmd
}

func NewRepoCreateCommand() *cobra.Command {
	var (
		provider string
		owner    string
		repo     string
		token    string
		public   bool
		host     string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new gitops repository",
		Example: util.Doc(`
# To run this command you need to create a personal access token for your git provider
# and provide it using:
	
    export GIT_TOKEN=<token>

# or with the flag:
	
    --token <token>

# Create a new gitops repository on github
    
    <BIN> repo create --owner foo --name bar --token abc123

# Create a public gitops repository on github
    
    <BIN> repo create --owner foo --name bar --token abc123 --public
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunRepoCreate(cmd.Context(), &RepoCreateOptions{
				Provider: provider,
				Owner:    owner,
				Repo:     repo,
				Token:    token,
				Public:   public,
				Host:     host,
			})
		},
	}

	util.Die(viper.BindEnv("git-token", "GIT_TOKEN"))

	cmd.Flags().StringVarP(&provider, "provider", "p", "github", "The git provider, "+fmt.Sprintf("one of: %v", strings.Join(supportedProviders, "|")))
	cmd.Flags().StringVarP(&owner, "owner", "o", "", "The name of the owner or organiaion")
	cmd.Flags().StringVarP(&repo, "name", "r", "", "The name of the repository")
	cmd.Flags().StringVarP(&token, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
	cmd.Flags().StringVar(&host, "host", "", "The git provider address (for on-premise git providers)")
	cmd.Flags().BoolVar(&public, "public", false, "If true, will create the repository as public (default is false)")

	util.Die(cmd.MarkFlagRequired("owner"))
	util.Die(cmd.MarkFlagRequired("name"))
	util.Die(cmd.MarkFlagRequired("git-token"))

	return cmd
}

func NewRepoBootstrapCommand() *cobra.Command {
	var (
		namespaced       bool
		dryRun           bool
		hidePassword     bool
		installationMode string
		f                kube.Factory
		appOptions       *application.CreateOptions
		repoOpts         *git.CloneOptions
	)

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap a new installation",
		Example: util.Doc(`
# To run this command you need to create a personal access token for your git provider
# and provide it using:
	
    export GIT_TOKEN=<token>

# or with the flag:
	
    --token <token>
		
# Install argo-cd on the current kubernetes context in the argocd namespace
# and persists the bootstrap manifests to the root of gitops repository
	
	<BIN> repo bootstrap --repo https://github.com/example/repo

	# Install argo-cd on the current kubernetes context in the argocd namespace
	# and persists the bootstrap manifests to a specific folder in the gitops repository

	<BIN> repo bootstrap --repo https://github.com/example/repo --installation-path path/to/bootstrap/root
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunRepoBootstrap(cmd.Context(), &RepoBootstrapOptions{
				RepoURL:          cmd.Flag("repo").Value.String(),
				Revision:         cmd.Flag("revision").Value.String(),
				GitToken:         cmd.Flag("git-token").Value.String(),
				InstallationPath: cmd.Flag("installation-path").Value.String(),
				InstallationMode: installationMode,
				Namespace:        cmd.Flag("namespace").Value.String(),
				KubeContext:      cmd.Flag("context").Value.String(),
				Namespaced:       namespaced,
				DryRun:           dryRun,
				HidePassword:     hidePassword,
				Timeout:          util.MustParseDuration(cmd.Flag("request-timeout").Value.String()),
				FS:               fs.Create(memfs.New()),
				KubeFactory:      f,
				AppOptions:       appOptions,
				CloneOptions:     repoOpts,
			})
		},
	}

	util.Die(viper.BindEnv("git-token", "GIT_TOKEN"))
	util.Die(viper.BindEnv("repo", "GIT_REPO"))

	cmd.Flags().BoolVar(&namespaced, "namespaced", false, "If true, install a namespaced version of argo-cd (no need for cluster-role)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "If true, print manifests instead of applying them to the cluster (nothing will be commited to git)")
	cmd.Flags().BoolVar(&hidePassword, "hide-password", false, "If true, will not print initial argo cd password")
	cmd.Flags().StringVar(&installationMode, "installation-mode", "normal", "One of: normal|flat. "+
		"If flat, will commit the bootstrap manifests, otherwise will commit the bootstrap kustomization.yaml")

	// add application flags
	appOptions = application.AddFlags(cmd)
	repoOpts, err := git.AddFlags(cmd)
	util.Die(err)

	// add kubernetes flags
	f = kube.AddFlags(cmd.Flags())

	util.Die(cmd.MarkFlagRequired("repo"))
	util.Die(cmd.MarkFlagRequired("git-token"))

	return cmd
}

func RunRepoCreate(ctx context.Context, opts *RepoCreateOptions) error {
	p, err := git.NewProvider(&git.Options{
		Type: opts.Provider,
		Auth: &git.Auth{
			Username: "git",
			Password: opts.Token,
		},
		Host: opts.Host,
	})
	if err != nil {
		return err
	}

	log.G().Infof("creating repo: %s/%s", opts.Owner, opts.Repo)
	repoUrl, err := p.CreateRepository(ctx, &git.CreateRepoOptions{
		Owner:   opts.Owner,
		Name:    opts.Repo,
		Private: !opts.Public,
	})
	if err != nil {
		return err
	}
	log.G().Infof("repo created at: %s", repoUrl)

	return nil
}

func RunRepoBootstrap(ctx context.Context, opts *RepoBootstrapOptions) error {

	switch opts.InstallationMode {
	case installationModeFlat, installationModeNormal:
	default:
		return fmt.Errorf("unknown installation mode: %s", opts.InstallationMode)
	}

	namespace := opts.Namespace
	if namespace == "" {
		opts.Namespace = store.Default.ArgoCDNamespace
	}

	argocdPath := opts.FS.Join(store.Default.BootsrtrapDir, store.Default.ArgoCDName)
	opts.AppOptions.SrcPath = argocdPath
	opts.AppOptions.AppName = store.Default.ArgoCDName

	if opts.AppOptions.AppSpecifier == "" {
		opts.AppOptions.AppSpecifier = getBootstrapAppSpecifier(opts.Namespaced)
	}

	if _, err := os.Stat(opts.AppOptions.AppSpecifier); err == nil {
		log.G().Warnf("detected local bootstrap manifests, using 'flat' installation mode")
		opts.InstallationMode = installationModeFlat
	}

	opts.AppOptions.Namespace = opts.Namespace

	log.G().WithFields(log.Fields{
		"repoURL":      opts.RepoURL,
		"revision":     opts.Revision,
		"namespace":    opts.Namespace,
		"kube-context": opts.KubeContext,
	}).Debug("starting with options: ")

	bootstrapApp, err := opts.AppOptions.ParseBootstrap()
	util.Die(err, "failed to parse application from flags")

	bootstrapAppYAML := createApp(
		bootstrapApp,
		store.Default.BootsrtrapAppName,
		opts.Revision,
		store.Default.BootsrtrapDir,
		false,
	)
	rootAppYAML := createApp(
		bootstrapApp,
		store.Default.RootAppName,
		opts.Revision,
		store.Default.EnvsDir,
		false,
	)

	argoCDAppYAML := createApp(
		bootstrapApp,
		store.Default.ArgoCDName,
		opts.Revision,
		argocdPath,
		true,
	)

	repoCredsYAML := getRepoCredsSecret(opts.GitToken, opts.Namespace)

	bootstrapYAML, err := bootstrapApp.GenerateManifests()
	util.Die(err)

	if opts.DryRun {
		log.G().Printf("%s", util.JoinManifests(bootstrapYAML, repoCredsYAML, bootstrapAppYAML, argoCDAppYAML, rootAppYAML))
		os.Exit(0)
	}

	log.G().Infof("cloning repo: %s", opts.RepoURL)

	// clone GitOps repo
	r, err := opts.CloneOptions.Clone(ctx, opts.FS)
	util.Die(err)

	log.G().Infof("using revision: \"%s\", installation path: \"%s\"", opts.Revision, opts.InstallationPath)
	opts.FS.MkdirAll(opts.InstallationPath, os.ModeDir)
	opts.FS.ChrootOrDie(opts.InstallationPath)
	checkRepoPath(opts.FS)
	log.G().Debug("repository is ok")

	// apply built manifest to k8s cluster
	log.G().Infof("using context: \"%s\", namespace: \"%s\"", opts.KubeContext, opts.Namespace)
	log.G().Infof("applying bootstrap manifests to cluster...")
	util.Die(opts.KubeFactory.Apply(ctx, opts.Namespace, util.JoinManifests(bootstrapYAML, repoCredsYAML)))

	bootstrapKust, err := bootstrapApp.Kustomization()
	util.Die(err)

	bootstrapKustYAML, err := yaml.Marshal(bootstrapKust)
	util.Die(err)

	// write argocd manifests
	if opts.InstallationMode == installationModeNormal {
		opts.FS.WriteFile(opts.FS.Join(argocdPath, "kustomization.yaml"), bootstrapKustYAML)
	} else {
		opts.FS.WriteFile(opts.FS.Join(argocdPath, "install.yaml"), bootstrapYAML)
	}

	// write envs root app
	opts.FS.WriteFile(opts.FS.Join(store.Default.BootsrtrapDir, store.Default.RootAppName+".yaml"), rootAppYAML)

	// write argocd app
	opts.FS.WriteFile(opts.FS.Join(argocdPath+".yaml"), argoCDAppYAML)

	// write ./envs/Dummy
	opts.FS.WriteFile(opts.FS.Join(store.Default.EnvsDir, store.Default.DummyName), []byte{})

	// wait for argocd to be ready before applying argocd-apps
	stop := util.WithSpinner(ctx, "waiting for argo-cd to be ready")
	waitClusterReady(ctx, opts.KubeFactory, opts.Timeout, opts.Namespace)
	stop()

	// push results to repo
	log.G().Infof("pushing bootstrap manifests to repo")
	util.Die(r.Persist(ctx, &git.PushOptions{
		CommitMsg: "Autopilot Bootstrap at " + opts.InstallationPath,
	}))

	// apply "Argo-CD" Application that references "bootstrap/argo-cd"
	log.G().Infof("applying argo-cd bootstrap application")
	util.Die(opts.KubeFactory.Apply(ctx, opts.Namespace, util.JoinManifests(bootstrapAppYAML)))

	if !opts.HidePassword {
		printInitialPassword(ctx, opts.KubeFactory, opts.Namespace)
	}
	return nil
}

func checkRepoPath(fs fs.FS) {
	folders := []string{store.Default.BootsrtrapDir, store.Default.EnvsDir}
	for _, folder := range folders {
		if fs.ExistsOrDie(folder) {
			util.Die(fmt.Errorf("folder %s already exist in: %s", folder, fs.Join(fs.Root(), folder)))
		}
	}
}

func createApp(bootstrapApp application.BootstrapApplication, name, revision, srcPath string, noFinalizer bool) []byte {
	app := bootstrapApp.CreateApp(name, revision, srcPath)
	if noFinalizer {
		app.ObjectMeta.Finalizers = []string{}
	}
	data, err := yaml.Marshal(app)
	util.Die(err)

	return data
}

func waitClusterReady(ctx context.Context, f kube.Factory, timeout time.Duration, namespace string) {
	util.Die(f.Wait(ctx, &kube.WaitOptions{
		Interval: store.Default.WaitInterval,
		Timeout:  timeout,
		Resources: []kube.Resource{
			{
				Name:      "argocd-server",
				Namespace: namespace,
				WaitFunc: func(ctx context.Context, f kube.Factory, ns, name string) (bool, error) {
					cs, err := f.KubernetesClientSet()
					if err != nil {
						return false, err
					}

					d, err := cs.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
					if err != nil {
						return false, err
					}

					return d.Status.ReadyReplicas >= *d.Spec.Replicas, nil
				},
			},
		},
	}))
}

func getRepoCredsSecret(token, namespace string) []byte {
	res, err := yaml.Marshal(&v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      store.Default.RepoCredsSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"git_username": []byte(store.Default.GitUsername),
			"git_token":    []byte(token),
		},
	})
	util.Die(err)

	return res
}

func printInitialPassword(ctx context.Context, f kube.Factory, namespace string) {
	cs := f.KubernetesClientSetOrDie()
	secret, err := cs.CoreV1().Secrets(namespace).Get(ctx, "argocd-initial-admin-secret", metav1.GetOptions{})
	util.Die(err)

	passwd, ok := secret.Data["password"]
	if !ok {
		util.Die(fmt.Errorf("argocd initial password not found"))
	}

	log.G(ctx).Printf("\n")
	log.G(ctx).Infof("argocd initialized. password: %s", passwd)
	log.G(ctx).Infof("run:\n\n    kubectl port-forward -n %s svc/argocd-server 8080:80\n\n", namespace)
}

func parseInstallationMode(installationMode string) {
	switch installationMode {
	case installationModeFlat:
	case installationModeNormal:
	default:
		util.Die(fmt.Errorf("unknown installation mode: %s", installationMode))
	}
}

func getBootstrapAppSpecifier(namespaced bool) string {
	if namespaced {
		return store.Get().InstallationManifestsNamespacedURL
	}

	return store.Get().InstallationManifestsURL
}
