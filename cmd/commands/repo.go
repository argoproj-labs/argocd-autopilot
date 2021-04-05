package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argocd-autopilot/pkg/application"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/kube"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	billy "github.com/go-git/go-billy/v5"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var supportedProviders = []string{"github"}

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
		url           string
		path          string
		token         string
		namespaced    bool
		argocdContext string
		gitopsOnly    bool
		appName       string
		appUrl        string
		dryRun        bool
		f             kube.Factory
		appOptions    *application.CreateOptions
	)

	// TODO: remove this
	_ = argocdContext
	_ = gitopsOnly
	_ = appName
	_ = appUrl
	_ = dryRun
	_ = f

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap a new installation",
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			fs := memfs.New()
			ctx := cmd.Context()

			// cs := f.KubernetesClientSetOrDie()
			// ns, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			// util.Die(err)

			log.G(ctx).WithField("repo", url).Info("cloning repo")

			r, err := git.Clone(ctx, fs, &git.CloneOptions{
				URL: url,
				Auth: &git.Auth{
					Username: "blank",
					Password: token,
				},
			})
			util.Die(err)

			log.G(ctx).Debug("Cloned Repository")
			util.Die(checkRepoPath(fs, path))

			log.G(ctx).Debug("Repository is OK")

			if appOptions.AppSpecifier == "" {
				if namespaced {
					appOptions.AppSpecifier = store.Get().InstallationManifestsNamespacedURL
				} else {
					appOptions.AppSpecifier = store.Get().InstallationManifestsURL
				}
			}
			app := appOptions.ParseOrDie()
			data, err := app.GenerateManifests()
			util.Die(err)

			// apply built manifest to k8s cluster

			// save built manifest to "boostrap/argo-cd/manifests.yaml"
			err = writeFile(fs, "bootstrap/argo-cd/argo-cd.yaml", data)
			util.Die(err)

			// create an argo-cd Application called "Argo-CD" that references "bootstrap/argo-cd"
			argoApp := newArgoCDApplication("argo-cd", "argo-cd", url, "bootstrap/argo-cd")

			// apply argo-cd Application to k8s cluster
			// save argo-cd Application manifest to "boostrap/argo-cd.yaml"
			err = persistArgoCDApplication(fs, "bootstrap/argo-cd.yaml", argoApp)
			util.Die(err)

			// create and apply an argo-cd Application called "Autopilot-root" that references "envs"
			// save application manifest to "boostrap/autopilot-root.yaml"

			err = persistRepoOrDie(ctx, r, token)
			util.Die(err)
			log.G(ctx).Debug("Finished bootstrap")
		},
	}

	util.Die(viper.BindEnv("git-token", "GIT_TOKEN"))

	cmd.Flags().StringVar(&url, "url", "", "The gitops repository clone url")

	cmd.Flags().StringVar(&path, "path", "/", "The path where we create the installation files (defaults to the root of the repository")
	cmd.Flags().StringVarP(&token, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
	cmd.Flags().BoolVar(&namespaced, "namespaced", false, "If true we will install a namespaced version of argo-cd (no need for cluster-role)")

	// cmd.Flags().StringVarP(&argocdContext, "argocd-context", "h", "", "argocdContext")

	// add application flags
	appOptions = application.AddApplicationFlags(cmd.Flags())

	// add kubernetes flags
	f = kube.AddKubeConfigFlags(cmd.InheritedFlags())
	cmdutil.AddDryRunFlag(cmd)

	util.Die(cmd.MarkFlagRequired("url"))
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

func newArgoCDApplication(name, namespace, url, src string) *v1alpha1.Application {
	return &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "argo-autopilot",
				"app.kubernetes.io/name":       name,
			},
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Application",
		},
		Spec: v1alpha1.ApplicationSpec{
			Source: v1alpha1.ApplicationSource{
				RepoURL: url,
				Path:    src,
			},
			Destination: v1alpha1.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: namespace,
			},
			SyncPolicy: &v1alpha1.SyncPolicy{
				Automated: &v1alpha1.SyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
			},
		},
	}
}

func persistArgoCDApplication(fs billy.Filesystem, path string, app *v1alpha1.Application) error {
	data, err := yaml.Marshal(app)
	if err != nil {
		return err
	}

	return writeFile(fs, path, data)
}
