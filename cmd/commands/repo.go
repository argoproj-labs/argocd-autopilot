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

	argocdapp "github.com/argoproj/argo-cd/v2/pkg/apis/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
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
		private  bool
		host     string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Creates a new gitops repository",
		Example: `
# Create a new gitops repository on github
    
    autopilot repo create --owner foo --repo bar --token abc123

# Create a public gitops repository on github
    
    autopilot repo create --owner foo --repo bar --token abc123 --private=false
`,
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

			log.G().Printf("creating repo: %s/%s", owner, repo)
			repoUrl, err := p.CreateRepository(cmd.Context(), &git.CreateRepoOptions{
				Owner:   owner,
				Name:    repo,
				Private: private,
			})
			util.Die(err)

			log.G().Printf("repo created at: %s", repoUrl)
		},
	}

	util.Die(viper.BindEnv("git-token", "GIT_TOKEN"))

	cmd.Flags().StringVarP(&provider, "provider", "p", "github", "The git provider, "+fmt.Sprintf("one of: %v", strings.Join(supportedProviders, "|")))
	cmd.Flags().StringVarP(&owner, "owner", "o", "", "The name of the owner or organiaion")
	cmd.Flags().StringVarP(&repo, "repo", "r", "", "The name of the repository")
	cmd.Flags().StringVarP(&token, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
	cmd.Flags().StringVar(&host, "host", "", "The git provider address (for on-premise git providers)")
	cmd.Flags().BoolVar(&private, "private", true, "If false, will create the repository as private (default is true)")

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
		argocdContext    string
		appName          string
		appUrl           string
		f                kube.Factory
		appOptions       *application.CreateOptions
	)

	// TODO: remove this
	_ = argocdContext
	_ = appName
	_ = appUrl

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap a new installation",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				err        error
				repoURL    = util.MustGetString(cmd.Flags(), "repo")
				revision   = util.MustGetString(cmd.Flags(), "revision")
				namespace  = util.MustGetString(cmd.Flags(), "namespace")
				timeoutStr = util.MustGetString(cmd.Flags(), "request-timeout")
			)

			timeout, err := time.ParseDuration(timeoutStr)
			util.Die(err)

			fs := memfs.New()
			ctx := cmd.Context()

			bootstrapPath := fs.Join(installationPath, "bootstrap") // TODO: magic number
			appOptions.SrcPath = fs.Join(bootstrapPath, "argo-cd")  // TODO: magic number

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

			bootstarpApp := appOptions.ParseOrDie(true)
			rootAppYAML := createRootApp(namespace, repoURL, fs.Join(installationPath, "envs")) // TODO: magic number
			repoCredsYAML := getRepoCredsSecret(token, namespace)
			bootstrapYAML, err := bootstarpApp.GenerateManifests()
			util.Die(err)

			argoCDYAML, err := yaml.Marshal(bootstarpApp.ArgoCD())
			util.Die(err)

			if dryRun {
				log.G().Printf("%s", util.JoinManifests(bootstrapYAML, repoCredsYAML, argoCDYAML, rootAppYAML))
				os.Exit(0)
			}

			log.G(ctx).WithField("repo", repoURL).Info("cloning repo")

			// clone GitOps repo
			r, err := git.Clone(ctx, fs, &git.CloneOptions{
				URL:      repoURL,
				Revision: revision,
				Auth: &git.Auth{
					Username: "blank",
					Password: token,
				},
			})
			util.Die(err)

			checkRepoPath(fs, installationPath)
			log.G(ctx).Debug("repository is ok")

			// apply built manifest to k8s cluster
			util.Die(f.Apply(ctx, namespace, util.JoinManifests(bootstrapYAML, repoCredsYAML)))

			writeFile(fs, fs.Join(bootstrapPath, "argo-cd/manifests.yaml"), bootstrapYAML) // TODO: magic number
			writeFile(fs, fs.Join(bootstrapPath, "argo-cd.yaml"), argoCDYAML)              // TODO: magic number
			writeFile(fs, fs.Join(bootstrapPath, "root.yaml"), rootAppYAML)                // TODO: magic number
			writeFile(fs, fs.Join(installationPath, "envs", "DUMMY"), []byte{})            // TODO: magic number

			// wait for argocd to be ready before applying argocd-apps
			waitClusterReady(ctx, f, timeout, namespace)

			// push results to repo
			persistRepoOrDie(ctx, r, token, installationPath)

			// apply "Argo-CD" Application that references "bootstrap/argo-cd"
			util.Die(f.Apply(ctx, namespace, util.JoinManifests(argoCDYAML, rootAppYAML)))
		},
	}

	util.Die(viper.BindEnv("git-token", "GIT_TOKEN"))

	cmd.Flags().StringVar(&installationPath, "installation-path", "/", "The path where we create the installation files (defaults to the root of the repository")
	cmd.Flags().StringVarP(&token, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
	cmd.Flags().BoolVar(&namespaced, "namespaced", false, "If true, install a namespaced version of argo-cd (no need for cluster-role)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "If true, print manifests instead of applying them to the cluster (nothing will be commited to git)")

	// add application flags
	appOptions = application.AddFlags(cmd, "argo-cd")

	// add kubernetes flags
	f = kube.AddFlags(cmd.Flags())

	util.Die(cmd.MarkFlagRequired("repo"))
	util.Die(cmd.MarkFlagRequired("git-token"))

	return cmd
}

func validateProvider(provider string) {
	log := log.G()
	found := false

	for _, p := range supportedProviders {
		if p == provider {
			found = true
			break
		}
	}

	if !found {
		log.Fatalf("provider not supported: %v", provider)
	}
}

func checkRepoPath(fs billy.Filesystem, path string) {
	folders := []string{"bootstrap", "envs", "kustomize"}
	for _, folder := range folders {
		exists, err := util.Exists(fs, fs.Join(path, folder))
		util.Die(err)

		if exists {
			util.Die(fmt.Errorf("folder %s already exist", folder))
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

func persistRepoOrDie(ctx context.Context, r git.Repository, token, installationPath string) {
	util.Die(r.Add(ctx, "."))

	_, err := r.Commit(ctx, "Autopilot Bootstrap at "+installationPath)
	util.Die(err)

	util.Die(r.Push(ctx, &git.PushOptions{
		Auth: &git.Auth{
			Username: "blank",
			Password: token,
		},
	}))
}

func createRootApp(namespace, repoURL, srcPath string) []byte {
	app := &v1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: argocdapp.Group + "/v1alpha1",
			Kind:       argocdapp.ApplicationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "root",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "argo-autopilot", // TODO: magic number
				"app.kubernetes.io/name":       "root",           // TODO: magic number
			},
			Finalizers: []string{
				"resources-finalizer.argocd.argoproj.io",
			},
		},
		Spec: v1alpha1.ApplicationSpec{
			Source: v1alpha1.ApplicationSource{
				RepoURL: repoURL,
				Path:    srcPath, // TODO: magic number
			},
			Destination: v1alpha1.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: namespace,
			},
			SyncPolicy: &v1alpha1.SyncPolicy{
				Automated: &v1alpha1.SyncPolicyAutomated{
					SelfHeal: true,
					Prune:    true,
				},
			},
		},
	}

	data, err := yaml.Marshal(app)
	util.Die(err)

	return data
}

func waitClusterReady(ctx context.Context, f kube.Factory, timeout time.Duration, namespace string) {
	util.Die(f.Wait(ctx, &kube.WaitOptions{
		Interval: time.Second * 3, // TODO: magic number
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
			Name:      "autopilot-secret", // TODO: magic number
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"git_username": []byte("username"), // TODO: magic number
			"git_token":    []byte(token),
		},
	})
	util.Die(err)

	return res
}
