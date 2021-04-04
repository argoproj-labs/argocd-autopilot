package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	util.Die(viper.BindEnv("token", "GIT_TOKEN"))

	cmd.Flags().StringVarP(&provider, "provider", "p", "github", fmt.Sprintf("one of: %v", strings.Join(supportedProviders, "|")))
	cmd.Flags().StringVarP(&owner, "owner", "o", "", "owner or organization name")
	cmd.Flags().StringVarP(&repo, "repo", "r", "", "repository name")
	cmd.Flags().StringVarP(&token, "token", "t", "", "your git provider api token")
	cmd.Flags().StringVar(&host, "host", "", "git provider address (for on-premise git providers)")
	cmd.Flags().BoolVar(&private, "private", true, "create the repository as private")

	util.Die(cmd.MarkFlagRequired("owner"))
	util.Die(cmd.MarkFlagRequired("repo"))
	util.Die(cmd.MarkFlagRequired("token"))

	return cmd
}

func NewRepoBootstrapCommand() *cobra.Command {
	var (
		url           string
		path          string
		token         string
		secret        string
		namespaced    bool
		argocdContext string
		gitopsOnly    bool
		appName       string
		appUrl        string
		dryRun        bool
	)
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap a new installation",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()

			log.G(ctx).Debug("hello world")
		},
	}

	cmd.Flags().StringVarP(&url, "url", "u", "", "url")
	cmd.Flags().StringVarP(&path, "path", "p", "", "path")
	cmd.Flags().StringVarP(&token, "token", "t", "", "token")
	cmd.Flags().StringVarP(&secret, "secret", "s", "", "secret")
	cmd.Flags().BoolVarP(&namespaced, "namespaced", "n", false, "namespaced")
	cmd.Flags().StringVarP(&argocdContext, "argocdContext", "h", "", "argocdContext")
	cmd.Flags().BoolVarP(&gitopsOnly, "gitopsOnly", "g", false, "gitopsOnly")
	cmd.Flags().StringVarP(&appName, "appName", "a", "", "appName")
	cmd.Flags().StringVarP(&appUrl, "appUrl", "z", "", "appUrl")
	cmd.Flags().BoolVarP(&dryRun, "dryRun", "d", false, "dryRun")

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
