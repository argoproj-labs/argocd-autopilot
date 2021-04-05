package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var supportedProviders = []string{"github"}

const defaultNamespace = "argo-cd"

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
	_ = f

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap a new installation",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				err       error
				repoURL   = util.MustGetString(cmd.Flags(), "repo")
				revision  = util.MustGetString(cmd.Flags(), "revision")
				namespace = util.MustGetString(cmd.Flags(), "namespace")
			)

			fs := memfs.New()
			ctx := cmd.Context()

			// cs := f.KubernetesClientSetOrDie()
			// ns, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			// util.Die(err)

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

			bootstarpApp := appOptions.ParseOrDie()
			bootstarpApp.ArgoCD().ObjectMeta.Namespace = namespace // override "default" namespace
			bootstarpApp.ArgoCD().Spec.Destination.Server = "https://kubernetes.default.svc"
			bootstarpApp.ArgoCD().Spec.Source.Path = "/bootstrap/argo-cd"

			data, err := bootstarpApp.GenerateManifests()
			util.Die(err)

			// // create argo-cd Application called "Autopilot-root" that references "envs"
			rootApp := createRootApp(namespace, repoURL, installationPath)

			dryRun := util.MustGetString(cmd.Flags(), "dry-run")
			if dryRun == "server" || dryRun == "client" {
				argoCDYAML, err := yaml.Marshal(bootstarpApp.ArgoCD())
				util.Die(err)

				rootYAML, err := yaml.Marshal(rootApp)
				util.Die(err)

				fmt.Printf("%s\n---\n%s\n---\n%s", string(data), string(rootYAML), string(argoCDYAML))
				os.Exit(0)
			}

			// apply built manifest to k8s cluster
			log.G(ctx).WithField("repo", repoURL).Info("cloning repo")

			r, err := git.Clone(ctx, fs, &git.CloneOptions{
				URL:      repoURL,
				Revision: revision,
				Auth: &git.Auth{
					Username: "blank",
					Password: token,
				},
			})
			util.Die(err)

			log.G(ctx).Debug("Cloned Repository")
			util.Die(checkRepoPath(fs, installationPath))

			log.G(ctx).Debug("Repository is OK")

			// save built manifest to "boostrap/argo-cd/manifests.yaml"
			err = writeFile(fs, "bootstrap/argo-cd/manifests.yaml", data)
			util.Die(err)

			// // save argo-cd Application manifest to "boostrap/argo-cd.yaml"
			err = persistArgoCDApplication(fs, "bootstrap/argo-cd.yaml", bootstarpApp.ArgoCD())
			util.Die(err)

			// // apply "Autopilot-root" that references "envs"

			// // save application manifest to "boostrap/autopilot-root.yaml"
			err = persistArgoCDApplication(fs, "bootstrap/root.yaml", rootApp)
			util.Die(err)

			err = persistRepoOrDie(ctx, r, token)
			util.Die(err)
			log.G(ctx).Debug("Finished bootstrap")
		},
	}

	util.Die(viper.BindEnv("git-token", "GIT_TOKEN"))

	cmd.Flags().StringVar(&installationPath, "installation-path", "/", "The path where we create the installation files (defaults to the root of the repository")
	cmd.Flags().StringVarP(&token, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
	cmd.Flags().BoolVar(&namespaced, "namespaced", false, "If true we will install a namespaced version of argo-cd (no need for cluster-role)")

	// cmd.Flags().StringVarP(&argocdContext, "argocd-context", "h", "", "argocdContext")

	// add application flags
	appOptions = application.AddApplicationFlags(cmd, "argo-cd")

	// add kubernetes flags
	f, _ = kube.AddKubeConfigFlags(cmd.Flags())
	cmdutil.AddDryRunFlag(cmd)

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

func checkRepoPath(fs billy.Filesystem, path string) error {
	folders := []string{"bootstrap", "envs", "kustomize"}
	for _, folder := range folders {
		exists, err := util.Exists(fs, fs.Join(path, folder))
		if err != nil {
			return err
		}

		if exists {
			return fmt.Errorf("folder %s already exist", folder)
		}
	}

	return nil
}

func writeFile(fs billy.Filesystem, path string, data []byte) error {
	folder := filepath.Base(path)
	err := fs.MkdirAll(folder, os.ModeDir)
	if err != nil {
		return err
	}

	f, err := fs.Create(path)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	return err
}

func persistRepoOrDie(ctx context.Context, r git.Repository, token string) error {
	err := r.Add(ctx, ".")
	util.Die(err)

	_, err = r.Commit(ctx, "Added stuff")
	util.Die(err)

	return r.Push(ctx, &git.PushOptions{
		Auth: &git.Auth{
			Username: "blank",
			Password: token,
		},
	})
}

func persistArgoCDApplication(fs billy.Filesystem, path string, app *v1alpha1.Application) error {
	data, err := yaml.Marshal(app)
	if err != nil {
		return err
	}

	return writeFile(fs, path, data)
}

func createRootApp(namespace, repoURL, installationPath string) *v1alpha1.Application {
	return &v1alpha1.Application{
		TypeMeta: v1.TypeMeta{
			APIVersion: argocdapp.Group + "/v1alpha1",
			Kind:       argocdapp.ApplicationKind,
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      "root",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "argo-autopilot",
				"app.kubernetes.io/name":       "root",
			},
			Finalizers: []string{
				"resources-finalizer.argocd.argoproj.io",
			},
		},
		Spec: v1alpha1.ApplicationSpec{
			Source: v1alpha1.ApplicationSource{
				RepoURL: repoURL,
				Path:    filepath.Join(installationPath, "envs"),
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
}
