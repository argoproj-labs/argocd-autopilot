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
    
    <BIN> repo create --owner foo --repo bar --token abc123

# Create a public gitops repository on github
    
    <BIN> repo create --owner foo --repo bar --token abc123 --public
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
	cmd.Flags().StringVarP(&repo, "repo", "r", "", "The name of the repository")
	cmd.Flags().StringVarP(&token, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
	cmd.Flags().StringVar(&host, "host", "", "The git provider address (for on-premise git providers)")
	cmd.Flags().BoolVar(&public, "public", false, "If true, will create the repository as public (default is false)")

	util.Die(cmd.MarkFlagRequired("owner"))
	util.Die(cmd.MarkFlagRequired("repo"))
	util.Die(cmd.MarkFlagRequired("git-token"))

	return cmd
}

func NewRepoBootstrapCommand() *cobra.Command {
	var (
		installationPath string
		token            string
		namespaced       bool
		dryRun           bool
		hidePassword     bool
		f                kube.Factory
		appOptions       *application.CreateOptions
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
				err        error
				repoURL    = util.MustGetString(cmd.Flags(), "repo")
				revision   = util.MustGetString(cmd.Flags(), "revision")
				namespace  = util.MustGetString(cmd.Flags(), "namespace")
				context    = util.MustGetString(cmd.Flags(), "context")
				timeoutStr = util.MustGetString(cmd.Flags(), "request-timeout")
			)

			timeout, err := time.ParseDuration(timeoutStr)
			util.Die(err)

			fs := memfs.New()
			ctx := cmd.Context()

			bootstrapPath := fs.Join(installationPath, store.Common.BootsrtrapDir)
			appOptions.SrcPath = fs.Join(bootstrapPath, store.Common.ArgoCDName)

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

			appOptions.Namespace = namespace

			log.G().WithFields(log.Fields{
				"repoURL":      repoURL,
				"revision":     revision,
				"namespace":    namespace,
				"kube-context": context,
			}).Debug("starting with options: ")

			bootstarpApp := appOptions.ParseOrDie(true)
			rootAppYAML := createRootApp(namespace, repoURL, fs.Join(installationPath, store.Common.EnvsDir), revision)
			repoCredsYAML := getRepoCredsSecret(token, namespace)
			bootstrapYAML, err := bootstarpApp.GenerateManifests()
			util.Die(err)

			argoCDYAML, err := yaml.Marshal(bootstarpApp.ArgoCD())
			util.Die(err)

			if dryRun {
				log.G().Printf("%s", util.JoinManifests(bootstrapYAML, repoCredsYAML, argoCDYAML, rootAppYAML))
				os.Exit(0)
			}

			log.G().Infof("cloning repo: %s", repoURL)

			// clone GitOps repo
			r, err := git.Clone(ctx, &git.CloneOptions{
				URL:      repoURL,
				Revision: revision,
				Auth: &git.Auth{
					Username: "username",
					Password: token,
				},
				FS: fs,
			})
			util.Die(err)

			log.G().Infof("using revision: \"%s\", installation path: \"%s\"", revision, installationPath)
			checkRepoPath(fs, installationPath)
			log.G().Debug("repository is ok")

			// apply built manifest to k8s cluster
			log.G().Infof("using context: \"%s\", namespace: \"%s\"", context, namespace)
			log.G().Infof("applying bootstrap manifests to cluster...")
			util.Die(f.Apply(ctx, namespace, util.JoinManifests(bootstrapYAML, repoCredsYAML)))

			bootstrapKust, err := bootstarpApp.Kustomization()
			util.Die(err)

			writeFile(fs, fs.Join(bootstrapPath, "kustomization.yaml"), bootstrapKust)
			writeFile(fs, fs.Join(installationPath, store.Common.EnvsDir, store.Common.DummyName), []byte{})

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
			util.Die(f.Apply(ctx, namespace, util.JoinManifests(argoCDYAML, rootAppYAML)))

			if !hidePassword {
				printInitialPassword(ctx, f, namespace)
			}
		},
	}

	util.Die(viper.BindEnv("git-token", "GIT_TOKEN"))
	util.Die(viper.BindEnv("repo", "GIT_REPO"))

	cmd.Flags().StringVar(&installationPath, "installation-path", "", "The path where we create the installation files (defaults to the root of the repository")
	cmd.Flags().StringVarP(&token, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
	cmd.Flags().BoolVar(&namespaced, "namespaced", false, "If true, install a namespaced version of argo-cd (no need for cluster-role)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "If true, print manifests instead of applying them to the cluster (nothing will be commited to git)")
	cmd.Flags().BoolVar(&hidePassword, "hide-password", false, "If true, will not print initial argo cd password")

	// add application flags
	appOptions = application.AddFlags(cmd, "argo-cd")

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

func createRootApp(namespace, repoURL, srcPath, revision string) []byte {
	app := application.NewRootApp(namespace, repoURL, srcPath, revision)
	data, err := yaml.Marshal(app.ArgoCD())
	util.Die(err)

	return data
}

func waitClusterReady(ctx context.Context, f kube.Factory, timeout time.Duration, namespace string) {
	util.Die(f.Wait(ctx, &kube.WaitOptions{
		Interval: store.Common.WaitInterval,
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
			Name:      store.Common.SecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"git_username": []byte(store.Common.SecretName),
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
