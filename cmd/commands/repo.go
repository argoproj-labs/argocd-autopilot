package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/argoproj/argocd-autopilot/pkg/application"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/kube"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	"github.com/ghodss/yaml"
	billy "github.com/go-git/go-billy/v5"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var supportedProviders = []string{"github"}

const defaultNamespace = "argocd"

const (
	installationModeFlat   = "flat"
	installationModeNormal = "normal"
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
		Run: func(cmd *cobra.Command, args []string) {
			validateProvider(provider)

			p, err := git.NewProvider(&git.Options{
				Type: provider,
				Auth: &git.Auth{
					Username: "blank",
					Password: token,
				},
				Host: host,
			})
			util.Die(err)

			log.G().Infof("creating repo: %s/%s", owner, repo)
			repoUrl, err := p.CreateRepository(cmd.Context(), &git.CreateRepoOptions{
				Owner:   owner,
				Name:    repo,
				Private: !public,
			})
			util.Die(err)

			log.G().Infof("repo created at: %s", repoUrl)
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
		
# Installs argo-cd on the current kubernetes context in the argocd namespace
# and persists the bootstrap manifests in the gitops repository
	
	<BIN> repo bootstrap --repo https://github.com/example/repo
`),
		Run: func(cmd *cobra.Command, args []string) {
			var (
				err              error
				repoURL          = cmd.Flag("repo").Value.String()
				gitToken         = cmd.Flag("git-token").Value.String()
				installationPath = cmd.Flag("installation-path").Value.String()
				revision         = cmd.Flag("revision").Value.String()
				namespace        = cmd.Flag("namespace").Value.String()
				context          = cmd.Flag("context").Value.String()
				timeout          = util.MustParseDuration(cmd.Flag("request-timeout").Value.String())
				fs               = memfs.New()
				ctx              = cmd.Context()
			)

			parseInstallationMode(installationMode)

			bootstrapPath := fs.Join(installationPath, store.Default.BootsrtrapDir)
			argocdPath := fs.Join(bootstrapPath, store.Default.ArgoCDName)
			envsPath := fs.Join(installationPath, store.Default.EnvsDir)
			appOptions.SrcPath = argocdPath

			if namespace == "" {
				namespace = defaultNamespace
			}

			if appOptions.AppSpecifier == "" {
				if namespaced {
					appOptions.AppSpecifier = store.Get().InstallationManifestsNamespacedURL
				} else {
					appOptions.AppSpecifier = store.Get().InstallationManifestsURL
				}
			}

			if _, err := os.Stat(appOptions.AppSpecifier); err == nil {
				log.G().Warnf("detected local bootstrap manifests, using 'flat' installation mode")
				installationMode = installationModeFlat
			}

			appOptions.Namespace = namespace

			log.G().WithFields(log.Fields{
				"repoURL":      repoURL,
				"revision":     revision,
				"namespace":    namespace,
				"kube-context": context,
			}).Debug("starting with options: ")

			bootstrapApp, err := appOptions.ParseBootstrap()
			util.Die(err, "failed to parse application from flags")

			bootstrapAppYAML := createApp(
				bootstrapApp,
				store.Default.BootsrtrapAppName,
				revision,
				bootstrapPath,
			)
			rootAppYAML := createApp(
				bootstrapApp,
				store.Default.RootAppName,
				revision,
				envsPath,
			)

			argoCDAppYAML := createApp(
				bootstrapApp,
				store.Default.ArgoCDName,
				revision,
				argocdPath,
			)

			repoCredsYAML := getRepoCredsSecret(gitToken, namespace)

			bootstrapYAML, err := bootstrapApp.GenerateManifests()
			util.Die(err)

			if dryRun {
				log.G().Printf("%s", util.JoinManifests(bootstrapYAML, repoCredsYAML, bootstrapAppYAML, argoCDAppYAML, rootAppYAML))
				os.Exit(0)
			}

			log.G().Infof("cloning repo: %s", repoURL)

			// clone GitOps repo
			r, err := repoOpts.Clone(ctx, fs)
			util.Die(err)

			log.G().Infof("using revision: \"%s\", installation path: \"%s\"", revision, installationPath)
			checkRepoPath(fs, installationPath)
			log.G().Debug("repository is ok")

			// apply built manifest to k8s cluster
			log.G().Infof("using context: \"%s\", namespace: \"%s\"", context, namespace)
			log.G().Infof("applying bootstrap manifests to cluster...")
			util.Die(f.Apply(ctx, namespace, util.JoinManifests(bootstrapYAML, repoCredsYAML)))

			bootstrapKust, err := bootstrapApp.Kustomization()
			util.Die(err)

			bootstrapKustYAML, err := yaml.Marshal(bootstrapKust)
			util.Die(err)

			// write argocd manifests
			if installationMode == installationModeNormal {
				writeFile(fs, fs.Join(argocdPath, "kustomization.yaml"), bootstrapKustYAML)
			} else {
				writeFile(fs, fs.Join(argocdPath, "install.yaml"), bootstrapYAML)
			}

			// write envs root app
			writeFile(fs, fs.Join(bootstrapPath, store.Default.RootAppName+".yaml"), rootAppYAML)

			// write argocd app
			writeFile(fs, fs.Join(bootstrapPath, store.Default.ArgoCDName+".yaml"), argoCDAppYAML)

			// write ./envs/Dummy
			writeFile(fs, fs.Join(envsPath, store.Default.DummyName), []byte{})

			// wait for argocd to be ready before applying argocd-apps
			stop := util.WithSpinner(ctx, "waiting for argo-cd to be ready")
			waitClusterReady(ctx, f, timeout, namespace)
			stop()

			// push results to repo
			log.G().Infof("pushing bootstrap manifests to repo")
			util.Die(r.Persist(ctx, &git.PushOptions{
				CommitMsg: "Autopilot Bootstrap at " + installationPath,
			}))

			// apply "Argo-CD" Application that references "bootstrap/argo-cd"
			log.G().Infof("applying argo-cd bootstrap application")
			util.Die(f.Apply(ctx, namespace, util.JoinManifests(bootstrapAppYAML)))

			if !hidePassword {
				printInitialPassword(ctx, f, namespace)
			}
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
	appOptions = application.AddFlags(cmd, "argo-cd")
	repoOpts, err := git.AddFlags(cmd)
	util.Die(err)

	// add kubernetes flags
	f = kube.AddFlags(cmd.Flags())

	util.Die(cmd.MarkFlagRequired("repo"))
	util.Die(cmd.MarkFlagRequired("git-token"))

	return cmd
}

func validateProvider(provider string) {
	for _, p := range supportedProviders {
		if p == provider {
			return
		}
	}

	log.G().Fatalf("provider not supported: %v", provider)
}

func checkRepoPath(fs billy.Filesystem, path string) {
	folders := []string{"bootstrap", "envs", "kustomize"}
	for _, folder := range folders {
		exists, err := util.Exists(fs, fs.Join(path, folder))
		util.Die(err)

		if exists {
			util.Die(fmt.Errorf("folder %s already exist in: %s", folder, fs.Join(path, folder)))
		}
	}
}

func writeFile(fs billy.Filesystem, path string, data []byte) {
	folder := filepath.Base(path)
	util.Die(fs.MkdirAll(folder, os.ModeDir))

	f, err := fs.Create(path)
	util.Die(err)

	_, err = f.Write(data)
	util.Die(err)
}

func createApp(bootstrapApp application.BootstrapApplication, name, revision, srcPath string) []byte {
	app := bootstrapApp.CreateApp(name, revision, srcPath)
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
