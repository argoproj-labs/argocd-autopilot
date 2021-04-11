package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/application"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/ghodss/yaml"
	"github.com/go-git/go-billy/v5"
	memfs "github.com/go-git/go-billy/v5/memfs"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewAppCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "application",
		Aliases: []string{"app"},
		Short:   "Manage applications",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
			os.Exit(1)
		},
	}

	cmd.AddCommand(NewAppCreateCommand())

	return cmd
}

func NewAppCreateCommand() *cobra.Command {
	var (
		installationPath string
		token            string
		envName          string
		appOptions       *application.CreateOptions
	)

	cmd := &cobra.Command{
		Use:   "create [APPNAME]",
		Short: "Add an application to an environment",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var (
				repoURL  = util.MustGetString(cmd.Flags(), "repo")
				revision = util.MustGetString(cmd.Flags(), "revision")
			)

			ctx := cmd.Context()
			fs := memfs.New()

			appName := args[0]

			log.G().WithFields(log.Fields{
				"repoURL":  repoURL,
				"revision": revision,
				"appName":  appName,
			}).Debug("starting with options: ")

			// clone repo
			log.G().Infof("cloning git repository: %s", repoURL)
			r, err := git.Clone(ctx, &git.CloneOptions{
				URL:      repoURL,
				FS:       fs,
				Revision: revision,
				Auth: &git.Auth{
					Username: "username",
					Password: token,
				},
			})
			util.Die(err)

			log.G().Infof("using installation path: %s", installationPath)
			fs = util.MustChroot(fs, installationPath)

			app := appOptions.ParseOrDie(false)

			// get application files
			basePath := fs.Join(store.Common.KustomizeDir, appName, "base", "kustomization.yaml")
			baseYAML, err := yaml.Marshal(app.Base())
			util.Die(err, "failed to marshal app base kustomization")

			overlayPath := fs.Join(store.Common.KustomizeDir, appName, "overlays", envName, "kustomization.yaml")
			overlayYAML, err := yaml.Marshal(app.Overlay())
			util.Die(err, "failed to marshal app overlay kustomization")

			configJSONPath := fs.Join(store.Common.KustomizeDir, appName, "overlays", envName, "config.json")
			configJSON, err := json.Marshal(app.ConfigJson())
			util.Die(err, "failed to marshal app config.json")

			// Create Base
			log.G().Debugf("checking if application base already exists: %s", basePath)
			if exists := checkExistsOrWriteFile(fs, basePath, baseYAML); !exists {
				log.G().Infof("created application base file at: %s", basePath)
			} else {
				log.G().Infof("application base file exists on: %s", basePath)
			}

			// Create Overlay
			log.G().Debugf("checking if application overlay already exists: %s", overlayPath)
			if exists := checkExistsOrWriteFile(fs, overlayPath, overlayYAML); !exists {
				log.G().Infof("created application overlay file at: %s", overlayPath)
			} else {
				// application already exists
				log.G().Infof("app \"%s\" already installed on env: %s", appName, envName)
				log.G().Infof("found overlay on: %s", overlayPath)
				os.Exit(1)
			}

			// Create config.json
			if exists := checkExistsOrWriteFile(fs, configJSONPath, configJSON); !exists {
				log.G().Infof("created application config file at: %s", configJSONPath)
			} else {
				log.G().Infof("application base file exists on: %s", configJSONPath)
			}

			commitMsg := fmt.Sprintf("installed app \"%s\" on environment \"%s\"", appName, envName)
			if installationPath != "" {
				commitMsg += fmt.Sprintf(" installation-path: %s", installationPath)
			}

			log.G().Info("committing changes to gitops repo...")
			util.Die(r.Persist(ctx, &git.PushOptions{CommitMsg: commitMsg}), "failed to push to repo")
			log.G().Info("application installed!")
		},
	}

	util.Die(viper.BindEnv("git-token", "GIT_TOKEN"))
	util.Die(viper.BindEnv("repo", "GIT_REPO"))

	cmd.Flags().StringVar(&installationPath, "installation-path", "", "The path where we create the installation files (defaults to the root of the repository")
	cmd.Flags().StringVarP(&token, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
	cmd.Flags().StringVar(&envName, "environment", "", "Environment name")

	appOptions = application.AddFlags(cmd, "")

	util.Die(cmd.MarkFlagRequired("repo"))
	util.Die(cmd.MarkFlagRequired("git-token"))

	return cmd
}

func checkExistsOrWriteFile(fs billy.Filesystem, path string, data []byte) bool {
	exists, err := util.Exists(fs, path)
	util.Die(err, fmt.Sprintf("failed to check if file exists on repo: %s", path))

	if exists {
		log.G().Debugf("file already exists: %s", path)
		return true
	}

	f, err := fs.Create(path)
	util.Die(err, fmt.Sprintf("failed to create file at: %s", path))

	_, err = f.Write(data)
	util.Die(err, fmt.Sprintf("failed to write to file: %s", path))
	return false
}
