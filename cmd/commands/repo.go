package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/argoproj/argocd-autopilot/pkg/application"
	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/kube"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	argocdv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdsettings "github.com/argoproj/argo-cd/v2/util/settings"
	"github.com/ghodss/yaml"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kusttypes "sigs.k8s.io/kustomize/api/types"
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
		AppSpecifier     string
		InstallationMode string
		Namespace        string
		KubeContext      string
		Namespaced       bool
		DryRun           bool
		HidePassword     bool
		Timeout          time.Duration
		FS               fs.FS
		KubeFactory      kube.Factory
		CloneOptions     *git.CloneOptions
	}

	bootstrapManifests struct {
		bootstrapApp           []byte
		rootApp                []byte
		argocdApp              []byte
		repoCreds              []byte
		applyManifests         []byte
		bootstrapKustomization []byte
		namespace              []byte
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

	die(viper.BindEnv("git-token", "GIT_TOKEN"))

	cmd.Flags().StringVarP(&provider, "provider", "p", "github", "The git provider, "+fmt.Sprintf("one of: %v", strings.Join(supportedProviders, "|")))
	cmd.Flags().StringVarP(&owner, "owner", "o", "", "The name of the owner or organiaion")
	cmd.Flags().StringVarP(&repo, "name", "r", "", "The name of the repository")
	cmd.Flags().StringVarP(&token, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
	cmd.Flags().StringVar(&host, "host", "", "The git provider address (for on-premise git providers)")
	cmd.Flags().BoolVar(&public, "public", false, "If true, will create the repository as public (default is false)")

	die(cmd.MarkFlagRequired("owner"))
	die(cmd.MarkFlagRequired("name"))
	die(cmd.MarkFlagRequired("git-token"))

	return cmd
}

func NewRepoBootstrapCommand() *cobra.Command {
	var (
		appSpecifier     string
		namespaced       bool
		dryRun           bool
		hidePassword     bool
		installationMode string
		f                kube.Factory
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
				AppSpecifier:     appSpecifier,
				InstallationMode: installationMode,
				Namespace:        cmd.Flag("namespace").Value.String(),
				KubeContext:      cmd.Flag("context").Value.String(),
				Namespaced:       namespaced,
				DryRun:           dryRun,
				HidePassword:     hidePassword,
				Timeout:          util.MustParseDuration(cmd.Flag("request-timeout").Value.String()),
				FS:               fs.Create(memfs.New()),
				KubeFactory:      f,
				CloneOptions:     repoOpts,
			})
		},
	}

	die(viper.BindEnv("git-token", "GIT_TOKEN"))
	die(viper.BindEnv("repo", "GIT_REPO"))

	cmd.Flags().StringVar(&appSpecifier, "app", "", "The application specifier (e.g. argocd@v1.0.2)")
	cmd.Flags().BoolVar(&namespaced, "namespaced", false, "If true, install a namespaced version of argo-cd (no need for cluster-role)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "If true, print manifests instead of applying them to the cluster (nothing will be commited to git)")
	cmd.Flags().BoolVar(&hidePassword, "hide-password", false, "If true, will not print initial argo cd password")
	cmd.Flags().StringVar(&installationMode, "installation-mode", "normal", "One of: normal|flat. "+
		"If flat, will commit the bootstrap manifests, otherwise will commit the bootstrap kustomization.yaml")

	// add application flags
	repoOpts, err := git.AddFlags(cmd)
	die(err)

	// add kubernetes flags
	f = kube.AddFlags(cmd.Flags())

	die(cmd.MarkFlagRequired("repo"))
	die(cmd.MarkFlagRequired("git-token"))

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
	var (
		err error
		r   git.Repository
	)
	if opts, err = setBootstrapOptsDefaults(*opts); err != nil {
		return err
	}

	log.G().WithFields(log.Fields{
		"repo-url":     opts.CloneOptions.URL,
		"revision":     opts.CloneOptions.Revision,
		"namespace":    opts.Namespace,
		"kube-context": opts.KubeContext,
	}).Debug("starting with options: ")

	manifests, err := buildBootstrapManifests(
		opts.Namespace,
		opts.AppSpecifier,
		opts.CloneOptions,
	)
	if err != nil {
		return fmt.Errorf("failed to build bootstrap manifests: %w", err)
	}

	// Dry Run check
	if opts.DryRun {
		fmt.Printf("%s", util.JoinManifests(
			manifests.applyManifests,
			manifests.repoCreds,
			manifests.bootstrapApp,
			manifests.argocdApp,
			manifests.rootApp,
		))
		os.Exit(0)
	}

	log.G().Infof("cloning repo: %s", opts.CloneOptions.URL)

	// clone GitOps repo
	r, opts.FS, err = opts.CloneOptions.Clone(ctx, opts.FS)
	if err != nil {
		return err
	}

	log.G().Infof("using revision: \"%s\", installation path: \"%s\"", opts.CloneOptions.Revision, opts.CloneOptions.RepoRoot)
	if err = validateRepo(opts.FS); err != nil {
		return err
	}

	log.G().Debug("repository is ok")

	// apply built manifest to k8s cluster
	log.G().Infof("using context: \"%s\", namespace: \"%s\"", opts.KubeContext, opts.Namespace)
	log.G().Infof("applying bootstrap manifests to cluster...")
	if err = opts.KubeFactory.Apply(ctx, opts.Namespace, util.JoinManifests(manifests.applyManifests, manifests.repoCreds)); err != nil {
		return fmt.Errorf("failed to apply bootstrap manifests to cluster: %w", err)
	}

	// write argocd manifests
	if err = writeManifestsToRepo(opts.FS, manifests, opts.InstallationMode); err != nil {
		return fmt.Errorf("failed to write manifests to repo: %w", err)
	}

	// wait for argocd to be ready before applying argocd-apps
	stop := util.WithSpinner(ctx, "waiting for argo-cd to be ready")

	if err = waitClusterReady(ctx, opts.KubeFactory, opts.Timeout, opts.Namespace); err != nil {
		return err
	}
	stop()

	// push results to repo
	log.G().Infof("pushing bootstrap manifests to repo")
	if err = r.Persist(ctx, &git.PushOptions{CommitMsg: "Autopilot Bootstrap at " + opts.CloneOptions.RepoRoot}); err != nil {
		return err
	}

	// apply "Argo-CD" Application that references "bootstrap/argo-cd"
	log.G().Infof("applying argo-cd bootstrap application")
	if err = opts.KubeFactory.Apply(ctx, opts.Namespace, manifests.bootstrapApp); err != nil {
		return err
	}

	if !opts.HidePassword {
		return printInitialPassword(ctx, opts.KubeFactory, opts.Namespace)
	}

	return nil
}

func setBootstrapOptsDefaults(opts RepoBootstrapOptions) (*RepoBootstrapOptions, error) {
	var err error

	switch opts.InstallationMode {
	case installationModeFlat, installationModeNormal:
	default:
		return nil, fmt.Errorf("unknown installation mode: %s", opts.InstallationMode)
	}

	if opts.Namespace == "" {
		opts.Namespace = store.Default.ArgoCDNamespace
	}

	if opts.AppSpecifier == "" {
		opts.AppSpecifier = getBootstrapAppSpecifier(opts.Namespaced)
	}

	if _, err := os.Stat(opts.AppSpecifier); err == nil {
		log.G().Warnf("detected local bootstrap manifests, using 'flat' installation mode")
		opts.InstallationMode = installationModeFlat
	}

	if opts.KubeContext == "" {
		if opts.KubeContext, err = kube.CurrentContext(); err != nil {
			return nil, err
		}
	}

	return &opts, nil
}

func validateRepo(fs fs.FS) error {
	folders := []string{store.Default.BootsrtrapDir, store.Default.ProjectsDir}
	for _, folder := range folders {
		if fs.ExistsOrDie(folder) {
			return fmt.Errorf("folder %s already exist in: %s", folder, fs.Join(fs.Root(), folder))
		}
	}

	return nil
}

type createBootstrapAppOptions struct {
	name        string
	namespace   string
	repoURL     string
	revision    string
	srcPath     string
	noFinalizer bool
}

func createApp(opts *createBootstrapAppOptions) ([]byte, error) {
	app := &argocdv1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       argocdv1alpha1.ApplicationSchemaGroupVersionKind.Kind,
			APIVersion: argocdv1alpha1.ApplicationSchemaGroupVersionKind.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: opts.namespace,
			Name:      opts.name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": store.Default.ManagedBy,
				"app.kubernetes.io/name":       opts.name,
			},
			Finalizers: []string{
				"resources-finalizer.argocd.argoproj.io",
			},
		},
		Spec: argocdv1alpha1.ApplicationSpec{
			Project: "default",
			Source: argocdv1alpha1.ApplicationSource{
				RepoURL:        opts.repoURL,
				Path:           opts.srcPath,
				TargetRevision: opts.revision,
			},
			Destination: argocdv1alpha1.ApplicationDestination{
				Server:    store.Default.DestServer,
				Namespace: opts.namespace,
			},
			SyncPolicy: &argocdv1alpha1.SyncPolicy{
				Automated: &argocdv1alpha1.SyncPolicyAutomated{
					SelfHeal: true,
					Prune:    true,
				},
				SyncOptions: []string{
					"allowEmpty=true",
				},
			},
			IgnoreDifferences: []argocdv1alpha1.ResourceIgnoreDifferences{
				{
					Group: "argoproj.io",
					Kind:  "Application",
					JSONPointers: []string{
						"/status",
					},
				},
			},
		},
	}
	if opts.noFinalizer {
		app.ObjectMeta.Finalizers = []string{}
	}

	return yaml.Marshal(app)
}

func waitClusterReady(ctx context.Context, f kube.Factory, timeout time.Duration, namespace string) error {
	return f.Wait(ctx, &kube.WaitOptions{
		Interval: store.Default.WaitInterval,
		Timeout:  timeout,
		Resources: []kube.Resource{
			{
				Name:      "argocd-server",
				Namespace: namespace,
				WaitFunc:  kube.WaitDeploymentReady,
			},
		},
	})
}

func getRepoCredsSecret(token, namespace string) ([]byte, error) {
	return yaml.Marshal(&v1.Secret{
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
}

func printInitialPassword(ctx context.Context, f kube.Factory, namespace string) error {
	cs := f.KubernetesClientSetOrDie()
	secret, err := cs.CoreV1().Secrets(namespace).Get(ctx, "argocd-initial-admin-secret", metav1.GetOptions{})
	if err != nil {
		return err
	}

	passwd, ok := secret.Data["password"]
	if !ok {
		return fmt.Errorf("argocd initial password not found")
	}

	log.G(ctx).Printf("\n")
	log.G(ctx).Infof("argocd initialized. password: %s", passwd)
	log.G(ctx).Infof("run:\n\n    kubectl port-forward -n %s svc/argocd-server 8080:80\n\n", namespace)
	return nil
}

func getBootstrapAppSpecifier(namespaced bool) string {
	if namespaced {
		return store.Get().InstallationManifestsNamespacedURL
	}

	return store.Get().InstallationManifestsURL
}

func buildBootstrapManifests(namespace, appSpecifier string, cloneOpts *git.CloneOptions) (*bootstrapManifests, error) {
	var err error
	manifests := &bootstrapManifests{}

	manifests.bootstrapApp, err = createApp(&createBootstrapAppOptions{
		name:      store.Default.BootsrtrapAppName,
		namespace: namespace,
		repoURL:   cloneOpts.URL,
		revision:  cloneOpts.Revision,
		srcPath:   filepath.Join(cloneOpts.RepoRoot, store.Default.BootsrtrapDir),
	})
	if err != nil {
		return nil, err
	}

	manifests.rootApp, err = createApp(&createBootstrapAppOptions{
		name:      store.Default.RootAppName,
		namespace: namespace,
		repoURL:   cloneOpts.URL,
		revision:  cloneOpts.Revision,
		srcPath:   filepath.Join(cloneOpts.RepoRoot, store.Default.ProjectsDir),
	})
	if err != nil {
		return nil, err
	}

	manifests.argocdApp, err = createApp(&createBootstrapAppOptions{
		name:        store.Default.ArgoCDName,
		namespace:   namespace,
		repoURL:     cloneOpts.URL,
		revision:    cloneOpts.Revision,
		srcPath:     filepath.Join(cloneOpts.RepoRoot, store.Default.BootsrtrapDir, store.Default.ArgoCDName),
		noFinalizer: true,
	})
	if err != nil {
		return nil, err
	}

	k, err := createBootstrapKustomization(namespace, cloneOpts.URL, appSpecifier)
	if err != nil {
		return nil, err
	}

	ns := kube.GenerateNamespace(namespace)
	manifests.namespace, err = yaml.Marshal(ns)
	if err != nil {
		return nil, err
	}

	gen, err := application.GenerateManifests(k)
	if err != nil {
		return nil, err
	}
	manifests.applyManifests = util.JoinManifests(manifests.namespace, gen)

	manifests.repoCreds, err = getRepoCredsSecret(cloneOpts.Auth.Password, namespace)
	if err != nil {
		return nil, err
	}

	manifests.bootstrapKustomization, err = yaml.Marshal(k)
	if err != nil {
		return nil, err
	}

	return manifests, nil
}

func writeManifestsToRepo(repoFS fs.FS, manifests *bootstrapManifests, installationMode string) error {
	argocdPath := repoFS.Join(store.Default.BootsrtrapDir, store.Default.ArgoCDName)
	var err error
	if installationMode == installationModeNormal {
		_, err = repoFS.WriteFile(repoFS.Join(argocdPath, "kustomization.yaml"), manifests.bootstrapKustomization)
	} else {
		_, err = repoFS.WriteFile(repoFS.Join(argocdPath, "install.yaml"), manifests.applyManifests)
	}
	if err != nil {
		return err
	}

	// write namespace manifest
	if _, err = repoFS.WriteFile(repoFS.Join(argocdPath, "namespace.yaml"), manifests.namespace); err != nil {
		return err
	}

	// write envs root app
	if _, err = repoFS.WriteFile(repoFS.Join(store.Default.BootsrtrapDir, store.Default.RootAppName+".yaml"), manifests.rootApp); err != nil {
		return err
	}

	// write argocd app
	if _, err = repoFS.WriteFile(repoFS.Join(store.Default.BootsrtrapDir, store.Default.ArgoCDName+".yaml"), manifests.argocdApp); err != nil {
		return err
	}

	// write ./envs/Dummy
	if _, err = repoFS.WriteFile(repoFS.Join(store.Default.ProjectsDir, store.Default.DummyName), []byte{}); err != nil {
		return err
	}

	return nil
}

func createBootstrapKustomization(namespace, repoURL, appSpecifier string) (*kusttypes.Kustomization, error) {
	credsYAML, err := createCreds(repoURL)
	if err != nil {
		return nil, err
	}

	k := &kusttypes.Kustomization{
		Resources: []string{
			appSpecifier,
			"namespace.yaml",
		},
		TypeMeta: kusttypes.TypeMeta{
			APIVersion: kusttypes.KustomizationVersion,
			Kind:       kusttypes.KustomizationKind,
		},
		ConfigMapGenerator: []kusttypes.ConfigMapArgs{
			{
				GeneratorArgs: kusttypes.GeneratorArgs{
					Name:     "argocd-cm",
					Behavior: kusttypes.BehaviorMerge.String(),
					KvPairSources: kusttypes.KvPairSources{
						LiteralSources: []string{
							"repository.credentials=" + string(credsYAML),
						},
					},
				},
			},
		},
		Namespace: namespace,
	}

	k.FixKustomizationPostUnmarshalling()
	errs := k.EnforceFields()
	if len(errs) > 0 {
		return nil, fmt.Errorf("kustomization errors: %s", strings.Join(errs, "\n"))
	}
	k.FixKustomizationPreMarshalling()

	return k, nil
}

func createCreds(repoUrl string) ([]byte, error) {
	creds := []argocdsettings.Repository{
		{
			URL: repoUrl,
			UsernameSecret: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "autopilot-secret",
				},
				Key: "git_username",
			},
			PasswordSecret: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "autopilot-secret",
				},
				Key: "git_token",
			},
		},
	}

	return yaml.Marshal(creds)
}
