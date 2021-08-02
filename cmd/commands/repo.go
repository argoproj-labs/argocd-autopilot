package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/argoproj-labs/argocd-autopilot/pkg/application"
	"github.com/argoproj-labs/argocd-autopilot/pkg/argocd"
	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	fsutils "github.com/argoproj-labs/argocd-autopilot/pkg/fs/utils"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git"
	"github.com/argoproj-labs/argocd-autopilot/pkg/kube"
	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	appset "github.com/argoproj-labs/applicationset/api/v1alpha1"
	argocdcommon "github.com/argoproj/argo-cd/v2/common"
	argocdsettings "github.com/argoproj/argo-cd/v2/util/settings"
	"github.com/ghodss/yaml"
	"github.com/go-git/go-billy/v5/memfs"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kusttypes "sigs.k8s.io/kustomize/api/types"
)

const (
	installationModeFlat   = "flat"
	installationModeNormal = "normal"
)

// used for mocking
var (
	argocdLogin        = argocd.Login
	currentKubeContext = kube.CurrentContext
	runKustomizeBuild  = application.GenerateManifests
)

type (
	RepoBootstrapOptions struct {
		AppSpecifier     string
		InstallationMode string
		Namespace        string
		KubeConfig       string
		DryRun           bool
		HidePassword     bool
		Insecure         bool
		Timeout          time.Duration
		KubeFactory      kube.Factory
		CloneOptions     *git.CloneOptions
	}

	RepoUninstallOptions struct {
		Namespace    string
		Timeout      time.Duration
		CloneOptions *git.CloneOptions
		KubeFactory  kube.Factory
	}

	bootstrapManifests struct {
		bootstrapApp           []byte
		rootApp                []byte
		clusterResAppSet       []byte
		clusterResConfig       []byte
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
			exit(1)
		},
	}

	cmd.AddCommand(NewRepoBootstrapCommand())
	cmd.AddCommand(NewRepoUninstallCommand())

	return cmd
}

func NewRepoBootstrapCommand() *cobra.Command {
	var (
		appSpecifier     string
		dryRun           bool
		hidePassword     bool
		insecure         bool
		installationMode string
		cloneOpts        *git.CloneOptions
		f                kube.Factory
	)

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap a new installation",
		Example: util.Doc(`
# To run this command you need to create a personal access token for your git provider
# and provide it using:

    export GIT_TOKEN=<token>

# or with the flag:

    --git-token <token>

# Install argo-cd on the current kubernetes context in the argocd namespace
# and persists the bootstrap manifests to the root of gitops repository

	<BIN> repo bootstrap --repo https://github.com/example/repo

# Install argo-cd on the current kubernetes context in the argocd namespace
# and persists the bootstrap manifests to a specific folder in the gitops repository

	<BIN> repo bootstrap --repo https://github.com/example/repo/path/to/installation_root
`),
		PreRun: func(_ *cobra.Command, _ []string) { cloneOpts.Parse() },
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunRepoBootstrap(cmd.Context(), &RepoBootstrapOptions{
				AppSpecifier:     appSpecifier,
				InstallationMode: installationMode,
				Namespace:        cmd.Flag("namespace").Value.String(),
				KubeConfig:       cmd.Flag("kubeconfig").Value.String(),
				DryRun:           dryRun,
				HidePassword:     hidePassword,
				Insecure:         insecure,
				Timeout:          util.MustParseDuration(cmd.Flag("request-timeout").Value.String()),
				KubeFactory:      f,
				CloneOptions:     cloneOpts,
			})
		},
	}

	cmd.Flags().StringVar(&appSpecifier, "app", "", "The application specifier (e.g. github.com/argoproj-labs/argocd-autopilot/manifests?ref=v0.2.5), overrides the default installation argo-cd manifests")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "If true, print manifests instead of applying them to the cluster (nothing will be commited to git)")
	cmd.Flags().BoolVar(&hidePassword, "hide-password", false, "If true, will not print initial argo cd password")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "Run Argo-CD server without TLS")
	cmd.Flags().StringVar(&installationMode, "installation-mode", "normal", "One of: normal|flat. "+
		"If flat, will commit the bootstrap manifests, otherwise will commit the bootstrap kustomization.yaml")

	cloneOpts = git.AddFlags(cmd, &git.AddFlagsOptions{
		FS:               memfs.New(),
		CreateIfNotExist: true,
	})

	// add kubernetes flags
	f = kube.AddFlags(cmd.Flags())

	return cmd
}

func RunRepoBootstrap(ctx context.Context, opts *RepoBootstrapOptions) error {
	var err error

	if opts, err = setBootstrapOptsDefaults(*opts); err != nil {
		return err
	}

	kubeContext, err := currentKubeContext()
	if err != nil {
		return err
	}

	log.G(ctx).WithFields(log.Fields{
		"repo-url":     opts.CloneOptions.URL(),
		"revision":     opts.CloneOptions.Revision(),
		"namespace":    opts.Namespace,
		"kube-context": kubeContext,
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
			manifests.namespace,
			manifests.applyManifests,
			manifests.repoCreds,
			manifests.bootstrapApp,
			manifests.argocdApp,
			manifests.rootApp,
		))
		exit(0)
		return nil
	}

	log.G(ctx).Infof("cloning repo: %s", opts.CloneOptions.URL())

	// clone GitOps repo
	r, repofs, err := getRepo(ctx, opts.CloneOptions)
	if err != nil {
		return err
	}

	log.G(ctx).Infof("using revision: \"%s\", installation path: \"%s\"", opts.CloneOptions.Revision(), opts.CloneOptions.Path())
	if err = validateRepo(repofs); err != nil {
		return err
	}

	log.G(ctx).Debug("repository is ok")

	// apply built manifest to k8s cluster
	log.G(ctx).Infof("using context: \"%s\", namespace: \"%s\"", kubeContext, opts.Namespace)
	log.G(ctx).Infof("applying bootstrap manifests to cluster...")
	if err = opts.KubeFactory.Apply(ctx, opts.Namespace, util.JoinManifests(manifests.namespace, manifests.applyManifests, manifests.repoCreds)); err != nil {
		return fmt.Errorf("failed to apply bootstrap manifests to cluster: %w", err)
	}

	// write argocd manifests
	if err = writeManifestsToRepo(repofs, manifests, opts.InstallationMode, opts.Namespace); err != nil {
		return fmt.Errorf("failed to write manifests to repo: %w", err)
	}

	// wait for argocd to be ready before applying argocd-apps
	stop := util.WithSpinner(ctx, "waiting for argo-cd to be ready")

	if err = waitClusterReady(ctx, opts.KubeFactory, opts.Timeout, opts.Namespace); err != nil {
		stop()
		return err
	}

	stop()

	// push results to repo
	log.G(ctx).Infof("pushing bootstrap manifests to repo")
	commitMsg := "Autopilot Bootstrap"
	if opts.CloneOptions.Path() != "" {
		commitMsg = "Autopilot Bootstrap at " + opts.CloneOptions.Path()
	}

	if _, err = r.Persist(ctx, &git.PushOptions{CommitMsg: commitMsg}); err != nil {
		return err
	}

	// apply "Argo-CD" Application that references "bootstrap/argo-cd"
	log.G(ctx).Infof("applying argo-cd bootstrap application")
	if err = opts.KubeFactory.Apply(ctx, opts.Namespace, manifests.bootstrapApp); err != nil {
		return err
	}

	passwd, err := getInitialPassword(ctx, opts.KubeFactory, opts.Namespace)
	if err != nil {
		return err
	}

	log.G(ctx).Infof("running argocd login to initialize argocd config")
	err = argocdLogin(&argocd.LoginOptions{
		Namespace:  opts.Namespace,
		Username:   "admin",
		Password:   passwd,
		KubeConfig: opts.KubeConfig,
		Insecure:   opts.Insecure,
	})
	if err != nil {
		return err
	}

	if !opts.HidePassword {
		log.G(ctx).Printf("")
		log.G(ctx).Infof("argocd initialized. password: %s", passwd)
		log.G(ctx).Infof("run:\n\n    kubectl port-forward -n %s svc/argocd-server 8080:80\n\n", opts.Namespace)
	}

	return nil
}

func NewRepoUninstallCommand() *cobra.Command {
	var (
		cloneOpts *git.CloneOptions
		f         kube.Factory
	)

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstalls an installation",
		Example: util.Doc(`
# To run this command you need to create a personal access token for your git provider
# and provide it using:

    export GIT_TOKEN=<token>

# or with the flag:

    --git-token <token>

# Uninstall argo-cd from the current kubernetes context in the argocd namespace
# and delete all manifests rom the root of gitops repository

	<BIN> repo uninstall --repo https://github.com/example/repo

# Uninstall argo-cd from the current kubernetes context in the argocd namespace
# and delete all manifests from a specific folder in the gitops repository

	<BIN> repo uninstall --repo https://github.com/example/repo/path/to/installation_root
`),
		PreRun: func(_ *cobra.Command, _ []string) { cloneOpts.Parse() },
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunRepoUninstall(cmd.Context(), &RepoUninstallOptions{
				Namespace:    cmd.Flag("namespace").Value.String(),
				Timeout:      util.MustParseDuration(cmd.Flag("request-timeout").Value.String()),
				CloneOptions: cloneOpts,
				KubeFactory:  f,
			})
		},
	}

	cloneOpts = git.AddFlags(cmd, &git.AddFlagsOptions{
		FS: memfs.New(),
	})
	f = kube.AddFlags(cmd.Flags())

	return cmd
}

func RunRepoUninstall(ctx context.Context, opts *RepoUninstallOptions) error {
	var err error

	opts = setUninstallOptsDefaults(*opts)
	kubeContext, err := currentKubeContext()
	if err != nil {
		return err
	}

	log.G(ctx).WithFields(log.Fields{
		"repo-url":     opts.CloneOptions.URL(),
		"revision":     opts.CloneOptions.Revision(),
		"namespace":    opts.Namespace,
		"kube-context": kubeContext,
	}).Debug("starting with options: ")

	log.G(ctx).Infof("cloning repo: %s", opts.CloneOptions.URL())
	r, repofs, err := getRepo(ctx, opts.CloneOptions)
	if err != nil {
		return err
	}

	log.G(ctx).Debug("deleting files from repo")
	if err = deleteGitOpsFiles(repofs); err != nil {
		return err
	}

	log.G(ctx).Info("pushing changes to remote")
	revision, err := r.Persist(ctx, &git.PushOptions{CommitMsg: "Autopilot Uninstall"})
	if err != nil {
		return err
	}

	stop := util.WithSpinner(ctx, fmt.Sprintf("waiting for '%s' to be finish syncing", store.Default.BootsrtrapAppName))
	if err = waitAppSynced(ctx, opts.KubeFactory, opts.Timeout, store.Default.BootsrtrapAppName, opts.Namespace, revision, false); err != nil {
		se, ok := err.(*kerrors.StatusError)
		if !ok || se.ErrStatus.Reason != metav1.StatusReasonNotFound {
			stop()
			return err
		}
	}

	stop()
	log.G(ctx).Info("Deleting cluster resources")
	if err = deleteClusterResources(ctx, opts.KubeFactory, opts.Timeout); err != nil {
		return err
	}

	log.G(ctx).Debug("Deleting leftovers from repo")
	if err = billyUtils.RemoveAll(repofs, store.Default.BootsrtrapDir); err != nil {
		return err
	}

	log.G(ctx).Info("pushing final commit to remote")
	if _, err := r.Persist(ctx, &git.PushOptions{CommitMsg: "Autopilot Uninstall, deleted leftovers"}); err != nil {
		return err
	}

	return nil
}

func setBootstrapOptsDefaults(opts RepoBootstrapOptions) (*RepoBootstrapOptions, error) {
	switch opts.InstallationMode {
	case installationModeFlat, installationModeNormal:
	case "":
		opts.InstallationMode = installationModeNormal
	default:
		return nil, fmt.Errorf("unknown installation mode: %s", opts.InstallationMode)
	}

	if opts.Namespace == "" {
		opts.Namespace = store.Default.ArgoCDNamespace
	}

	if opts.AppSpecifier == "" {
		opts.AppSpecifier = getBootstrapAppSpecifier(opts.Insecure)
	}

	if _, err := os.Stat(opts.AppSpecifier); err == nil {
		log.G().Warnf("detected local bootstrap manifests, using 'flat' installation mode")
		opts.InstallationMode = installationModeFlat
	}

	return &opts, nil
}

func validateRepo(repofs fs.FS) error {
	folders := []string{store.Default.BootsrtrapDir, store.Default.ProjectsDir}
	for _, folder := range folders {
		if repofs.ExistsOrDie(folder) {
			return fmt.Errorf("folder %s already exist in: %s", folder, repofs.Join(repofs.Root(), folder))
		}
	}

	return nil
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
			Labels: map[string]string{
				store.Default.LabelKeyAppManagedBy: store.Default.LabelValueManagedBy,
			},
		},
		Data: map[string][]byte{
			"git_username": []byte(store.Default.GitUsername),
			"git_token":    []byte(token),
		},
	})
}

func getInitialPassword(ctx context.Context, f kube.Factory, namespace string) (string, error) {
	cs := f.KubernetesClientSetOrDie()
	secret, err := cs.CoreV1().Secrets(namespace).Get(ctx, "argocd-initial-admin-secret", metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	passwd, ok := secret.Data["password"]
	if !ok {
		return "", fmt.Errorf("argocd initial password not found")
	}

	return string(passwd), nil
}

func getBootstrapAppSpecifier(insecure bool) string {
	if insecure {
		return store.Get().InstallationManifestsInsecureURL
	}

	return store.Get().InstallationManifestsURL
}

func buildBootstrapManifests(namespace, appSpecifier string, cloneOpts *git.CloneOptions) (*bootstrapManifests, error) {
	var err error
	manifests := &bootstrapManifests{}

	manifests.bootstrapApp, err = createApp(&createAppOptions{
		name:      store.Default.BootsrtrapAppName,
		namespace: namespace,
		repoURL:   cloneOpts.URL(),
		revision:  cloneOpts.Revision(),
		srcPath:   filepath.Join(cloneOpts.Path(), store.Default.BootsrtrapDir),
	})
	if err != nil {
		return nil, err
	}

	manifests.rootApp, err = createApp(&createAppOptions{
		name:      store.Default.RootAppName,
		namespace: namespace,
		repoURL:   cloneOpts.URL(),
		revision:  cloneOpts.Revision(),
		srcPath:   filepath.Join(cloneOpts.Path(), store.Default.ProjectsDir),
	})
	if err != nil {
		return nil, err
	}

	manifests.argocdApp, err = createApp(&createAppOptions{
		name:        store.Default.ArgoCDName,
		namespace:   namespace,
		repoURL:     cloneOpts.URL(),
		revision:    cloneOpts.Revision(),
		srcPath:     filepath.Join(cloneOpts.Path(), store.Default.BootsrtrapDir, store.Default.ArgoCDName),
		noFinalizer: true,
	})
	if err != nil {
		return nil, err
	}

	manifests.clusterResAppSet, err = createAppSet(&createAppSetOptions{
		name:                        store.Default.ClusterResourcesDir,
		namespace:                   namespace,
		repoURL:                     cloneOpts.URL(),
		revision:                    cloneOpts.Revision(),
		appName:                     store.Default.ClusterResourcesDir + "-{{name}}",
		appNamespace:                namespace,
		destServer:                  "{{server}}",
		prune:                       false,
		preserveResourcesOnDeletion: true,
		srcPath:                     filepath.Join(cloneOpts.Path(), store.Default.BootsrtrapDir, store.Default.ClusterResourcesDir, "{{name}}"),
		generators: []appset.ApplicationSetGenerator{
			{
				Git: &appset.GitGenerator{
					RepoURL:  cloneOpts.URL(),
					Revision: cloneOpts.Revision(),
					Files: []appset.GitFileGeneratorItem{
						{
							Path: filepath.Join(
								cloneOpts.Path(),
								store.Default.BootsrtrapDir,
								store.Default.ClusterResourcesDir,
								"*.json",
							),
						},
					},
					RequeueAfterSeconds: &DefaultApplicationSetGeneratorInterval,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	manifests.clusterResConfig, err = json.Marshal(&application.ClusterResConfig{Name: store.Default.ClusterContextName, Server: store.Default.DestServer})
	if err != nil {
		return nil, err
	}

	k, err := createBootstrapKustomization(namespace, cloneOpts.URL(), appSpecifier)
	if err != nil {
		return nil, err
	}

	if namespace != "" && namespace != "default" {
		ns := kube.GenerateNamespace(namespace)
		manifests.namespace, err = yaml.Marshal(ns)
		if err != nil {
			return nil, err
		}
	}

	manifests.applyManifests, err = runKustomizeBuild(k)
	if err != nil {
		return nil, err
	}

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

func writeManifestsToRepo(repoFS fs.FS, manifests *bootstrapManifests, installationMode, namespace string) error {
	var bulkWrites []fsutils.BulkWriteRequest
	argocdPath := repoFS.Join(store.Default.BootsrtrapDir, store.Default.ArgoCDName)
	clusterResReadme := []byte(strings.ReplaceAll(string(clusterResReadmeTpl), "{CLUSTER}", store.Default.ClusterContextName))

	if installationMode == installationModeNormal {
		bulkWrites = []fsutils.BulkWriteRequest{
			{Filename: repoFS.Join(argocdPath, "kustomization.yaml"), Data: manifests.bootstrapKustomization},
		}
	} else {
		bulkWrites = []fsutils.BulkWriteRequest{
			{Filename: repoFS.Join(argocdPath, "install.yaml"), Data: manifests.applyManifests},
		}
	}

	bulkWrites = append(bulkWrites, []fsutils.BulkWriteRequest{
		{Filename: repoFS.Join(store.Default.BootsrtrapDir, store.Default.RootAppName+".yaml"), Data: manifests.rootApp},                                                    // write projects root app
		{Filename: repoFS.Join(store.Default.BootsrtrapDir, store.Default.ArgoCDName+".yaml"), Data: manifests.argocdApp},                                                   // write argocd app
		{Filename: repoFS.Join(store.Default.BootsrtrapDir, store.Default.ClusterResourcesDir+".yaml"), Data: manifests.clusterResAppSet},                                   // write cluster-resources appset
		{Filename: repoFS.Join(store.Default.BootsrtrapDir, store.Default.ClusterResourcesDir, store.Default.ClusterContextName, "README.md"), Data: clusterResReadme},      // write ./bootstrap/cluster-resources/in-cluster/README.md
		{Filename: repoFS.Join(store.Default.BootsrtrapDir, store.Default.ClusterResourcesDir, store.Default.ClusterContextName+".json"), Data: manifests.clusterResConfig}, // write ./bootstrap/cluster-resources/in-cluster.json
		{Filename: repoFS.Join(store.Default.ProjectsDir, "README.md"), Data: projectReadme},                                                                                // write ./projects/README.md
		{Filename: repoFS.Join(store.Default.AppsDir, "README.md"), Data: appsReadme},                                                                                       // write ./apps/README.md
	}...)

	if manifests.namespace != nil {
		// write ./bootstrap/cluster-resources/in-cluster/...-ns.yaml
		bulkWrites = append(
			bulkWrites,
			fsutils.BulkWriteRequest{Filename: repoFS.Join(store.Default.BootsrtrapDir, store.Default.ClusterResourcesDir, store.Default.ClusterContextName, namespace+"-ns.yaml"), Data: manifests.namespace},
		)
	}

	return fsutils.BulkWrite(repoFS, bulkWrites...)
}

func createBootstrapKustomization(namespace, repoURL, appSpecifier string) (*kusttypes.Kustomization, error) {
	credsYAML, err := createCreds(repoURL)
	if err != nil {
		return nil, err
	}

	k := &kusttypes.Kustomization{
		Resources: []string{
			appSpecifier,
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

	return k, k.FixKustomizationPreMarshalling()
}

func createCreds(repoUrl string) ([]byte, error) {
	host, _, _, _, _, _, _ := util.ParseGitUrl(repoUrl)
	creds := []argocdsettings.RepositoryCredentials{
		{
			URL: host,
			UsernameSecret: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: store.Default.RepoCredsSecretName,
				},
				Key: "git_username",
			},
			PasswordSecret: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: store.Default.RepoCredsSecretName,
				},
				Key: "git_token",
			},
		},
	}

	return yaml.Marshal(creds)
}

func setUninstallOptsDefaults(opts RepoUninstallOptions) *RepoUninstallOptions {
	if opts.Namespace == "" {
		opts.Namespace = store.Default.ArgoCDNamespace
	}

	return &opts
}

func deleteGitOpsFiles(repofs fs.FS) error {
	if err := billyUtils.RemoveAll(repofs, store.Default.AppsDir); err != nil {
		return fmt.Errorf("failed deleting '%s' folder: %w", store.Default.AppsDir, err)
	}

	if err := billyUtils.RemoveAll(repofs, store.Default.BootsrtrapDir); err != nil {
		return fmt.Errorf("failed deleting '%s' folder: %w", store.Default.BootsrtrapDir, err)
	}

	if err := billyUtils.RemoveAll(repofs, store.Default.ProjectsDir); err != nil {
		return fmt.Errorf("failed deleting '%s' folder: %w", store.Default.ProjectsDir, err)
	}

	if err := billyUtils.WriteFile(repofs, repofs.Join(store.Default.BootsrtrapDir, store.Default.DummyName), []byte{}, 0666); err != nil {
		return fmt.Errorf("failed creating '%s' file in '%s' folder: %w", store.Default.DummyName, store.Default.ProjectsDir, err)
	}

	return nil
}

func deleteClusterResources(ctx context.Context, f kube.Factory, timeout time.Duration) error {
	if err := f.Delete(ctx, &kube.DeleteOptions{
		LabelSelector: store.Default.LabelKeyAppManagedBy + "=" + store.Default.LabelValueManagedBy,
		ResourceTypes: []string{"applications", "secrets"},
		Timeout:       timeout,
	}); err != nil {
		return fmt.Errorf("failed deleting argocd-autopilot resources: %w", err)
	}

	if err := f.Delete(ctx, &kube.DeleteOptions{
		LabelSelector: argocdcommon.LabelKeyAppInstance + "=" + store.Default.ArgoCDName,
		ResourceTypes: []string{
			"all",
			"configmaps",
			"secrets",
			"serviceaccounts",
			"networkpolicies",
			"rolebindings",
			"roles",
		},
		Timeout: timeout,
	}); err != nil {
		return fmt.Errorf("failed deleting Argo-CD resources: %w", err)
	}

	return nil
}
