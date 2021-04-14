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
		envName    string
		appOptions *application.CreateOptions
		repoOpts   *git.CloneOptions
	)

	cmd := &cobra.Command{
		Use:   "create [APP_NAME]",
		Short: "Add an application to an environment",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				repoURL          = cmd.Flag("repo").Value.String()
				revision         = cmd.Flag("revision").Value.String()
				installationPath = cmd.Flag("installation-path").Value.String()

				ctx = cmd.Context()
				fs  = memfs.New()
			)

			if len(args) < 1 {
				log.G().Fatal("must enter application name")
			}
			appOptions.AppName = args[0]

			log.G().WithFields(log.Fields{
				"repoURL":  repoURL,
				"revision": revision,
				"appName":  appOptions.AppName,
			}).Debug("starting with options: ")

			// clone repo
			log.G().Infof("cloning git repository: %s", repoURL)
			r, err := repoOpts.Clone(ctx, fs)
			util.Die(err)

			log.G().Infof("using installation path: %s", installationPath)
			fs = util.MustChroot(fs, installationPath)

			util.MustExists(fs, store.Default.BootsrtrapDir, util.Doc(fmt.Sprintf("Bootstrap folder not found, please execute `<BIN> repo bootstrap --installation-path %s`", installationPath)))
			envExists := util.MustCheckEnvExists(fs, envName)
			if !envExists {
				log.G().Fatalf(util.Doc(fmt.Sprintf("env '%[1]s' not found, please execute `<BIN> env create %[1]s`", envName)))
			}

			log.G().Debug("repository is ok")

			app, err := appOptions.Parse()
			util.Die(err, "failed to parse application from flags")

			// get application files
			basePath := fs.Join(store.Default.KustomizeDir, appOptions.AppName, "base", "kustomization.yaml")
			baseYAML, err := yaml.Marshal(app.Base())
			util.Die(err, "failed to marshal app base kustomization")

			overlayPath := fs.Join(store.Default.KustomizeDir, appOptions.AppName, "overlays", envName, "kustomization.yaml")
			overlayYAML, err := yaml.Marshal(app.Overlay())
			util.Die(err, "failed to marshal app overlay kustomization")

			nsPath := fs.Join(store.Default.KustomizeDir, appOptions.AppName, "overlays", envName, "namespace.yaml")
			nsYAML, err := yaml.Marshal(app.Namespace())
			util.Die(err, "failed to marshal app overlay namespace")

			configJSONPath := fs.Join(store.Default.KustomizeDir, appOptions.AppName, "overlays", envName, "config.json")
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
				log.G().Infof("app \"%s\" already installed on env: %s", appOptions.AppName, envName)
				log.G().Infof("found overlay on: %s", overlayPath)
				os.Exit(1)
			}

			if exists := checkExistsOrWriteFile(fs, nsPath, nsYAML); !exists {
				log.G().Infof("created application namespace file at: %s", overlayPath)
			} else {
				// application already exists
				log.G().Infof("app \"%s\" already installed on env: %s", appOptions.AppName, envName)
				log.G().Infof("found overlay on: %s", overlayPath)
				os.Exit(1)
			}

			// Create config.json
			if exists := checkExistsOrWriteFile(fs, configJSONPath, configJSON); !exists {
				log.G().Infof("created application config file at: %s", configJSONPath)
			} else {
				log.G().Infof("application base file exists on: %s", configJSONPath)
			}

			commitMsg := fmt.Sprintf("installed app \"%s\" on environment \"%s\"", appOptions.AppName, envName)
			if installationPath != "" {
				commitMsg += fmt.Sprintf(" installation-path: %s", installationPath)
			}

			log.G().Info("committing changes to gitops repo...")
			util.Die(r.Persist(ctx, &git.PushOptions{CommitMsg: commitMsg}), "failed to push to repo")
			log.G().Info("application installed!")
		},
	}

	cmd.Flags().StringVar(&envName, "env", "", "Environment name")
	util.Die(cmd.MarkFlagRequired("env"))

	appOptions = application.AddFlags(cmd)
	util.Die(cmd.MarkFlagRequired("app"))
	repoOpts, err := git.AddFlags(cmd)
	util.Die(err)

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
