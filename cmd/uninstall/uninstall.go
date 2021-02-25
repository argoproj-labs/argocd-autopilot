package uninstall

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	envman "github.com/codefresh-io/cf-argo/pkg/environments-manager"
	cferrors "github.com/codefresh-io/cf-argo/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/pkg/client/clientset/versioned"
	"github.com/codefresh-io/cf-argo/pkg/git"
	"github.com/codefresh-io/cf-argo/pkg/kube"
	"github.com/codefresh-io/cf-argo/pkg/log"
	"github.com/codefresh-io/cf-argo/pkg/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

type options struct {
	repoURL  string
	envName  string
	gitToken string
	dryRun   bool
}

var values struct {
	GitopsRepoClonePath string
	GitopsRepo          git.Repository
	CommitRev           string
}

var renderValues struct {
	EnvName      string
	RepoURL      string
	RepoOwnerURL string
	GitToken     string
}

func New(ctx context.Context) *cobra.Command {
	var opts options

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstalls an Argo Enterprise solution from a specified cluster and installation",
		Long:  "This command will clear all Argo-CD managed resources relating to a specific installation, from a specific cluster",
		Run: func(cmd *cobra.Command, args []string) {
			fillValues(&opts)
			uninstall(ctx, &opts)
		},
	}

	// add kubernetes flags
	store.Get().KubeConfig.AddFlagSet(cmd)

	_ = viper.BindEnv("repo-url", "REPO_URL")
	_ = viper.BindEnv("env-name", "ENV_NAME")
	_ = viper.BindEnv("git-token", "GIT_TOKEN")
	viper.SetDefault("dry-run", false)

	cmd.Flags().StringVar(&opts.repoURL, "repo-url", viper.GetString("repo-url"), "the gitops repository url. If it does not exist we will try to create it for you [REPO_URL]")
	cmd.Flags().StringVar(&opts.envName, "env-name", viper.GetString("env-name"), "name of the Argo Enterprise environment to create")
	cmd.Flags().StringVar(&opts.gitToken, "git-token", viper.GetString("git-token"), "git token which will be used by argo-cd to create the gitops repository")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", viper.GetBool("dry-run"), "when true, the command will have no side effects, and will only output the manifests to stdout")

	cferrors.MustContext(ctx, cmd.MarkFlagRequired("repo-url"))
	cferrors.MustContext(ctx, cmd.MarkFlagRequired("env-name"))

	return cmd
}

func fillValues(opts *options) {
	var err error
	cferrors.CheckErr(err)

	renderValues.EnvName = opts.envName
}

func uninstall(ctx context.Context, opts *options) {
	defer func() {
		cleanup(ctx)
		if err := recover(); err != nil {
			panic(err)
		}
	}()

	cloneExistingRepo(ctx, opts)

	conf, err := envman.LoadConfig(values.GitopsRepoClonePath)
	cferrors.CheckErr(err)

	env, exists := conf.Environments[opts.envName]
	if !exists {
		panic(envman.ErrEnvironmentNotExist)
	}

	shouldClean, err := env.Uninstall()
	cferrors.CheckErr(err)

	persistGitopsRepo(ctx, opts, fmt.Sprintf("uninstalled environment %s", opts.envName))

	if shouldClean {
		rootApp, err := env.GetRootApp()
		cferrors.CheckErr(err)

		log.G(ctx).Printf("waiting for root application sync... (might take a few seconds)")
		awaitSync(ctx, opts, rootApp)

		log.G(ctx).Printf("deleting root application")
		deleteArgocdApp(ctx, opts, rootApp)

		log.G(ctx).Printf("cleaning up the repository")
		cferrors.CheckErr(conf.DeleteEnvironmentP(ctx, opts.envName, renderValues, opts.dryRun))

		persistGitopsRepo(ctx, opts, fmt.Sprintf("cleanup %s resources", opts.envName))

		log.G(ctx).Printf("all managed resources in '%s' have been removed, including argo-cd", opts.envName)
	} else {
		log.G(ctx).Printf("all managed resources in '%s' have been removed, argo-cd and user Applications remain on cluster", opts.envName)
	}
}

func cloneExistingRepo(ctx context.Context, opts *options) {
	p, err := git.NewProvider(&git.Options{
		Type: "github", // only option for now
		Auth: &git.Auth{
			Password: opts.gitToken,
		},
	})
	cferrors.CheckErr(err)

	values.GitopsRepo, err = p.CloneRepository(ctx, opts.repoURL)
	cferrors.CheckErr(err)

	values.GitopsRepoClonePath, err = values.GitopsRepo.Root()
	cferrors.CheckErr(err)
}

func persistGitopsRepo(ctx context.Context, opts *options, msg string) {
	var err error
	cferrors.CheckErr(values.GitopsRepo.Add(ctx, "."))

	values.CommitRev, err = values.GitopsRepo.Commit(ctx, msg)
	cferrors.CheckErr(err)

	if opts.dryRun {
		return
	}

	log.G(ctx).Printf("pushing to gitops repo...")
	err = values.GitopsRepo.Push(ctx, &git.PushOptions{
		Auth: &git.Auth{
			Password: opts.gitToken,
		},
	})
	cferrors.CheckErr(err)
}

func awaitSync(ctx context.Context, opts *options, app *envman.Application) {
	awaitAppCondition(ctx, opts, app, func(a *v1alpha1.Application, err error) (bool, error) {
		if err != nil {
			return false, err
		}

		return a.Status.Sync.Status == v1alpha1.SyncStatusCodeSynced && a.Status.Sync.Revision == values.CommitRev, nil
	})
}

func deleteArgocdApp(ctx context.Context, opts *options, app *envman.Application) {
	projData, err := ioutil.ReadFile(filepath.Join(filepath.Dir(app.Path), fmt.Sprintf("%s-project.yaml", opts.envName)))
	cferrors.CheckErr(err)

	appData, err := ioutil.ReadFile(app.Path)
	cferrors.CheckErr(err)

	manifests := []byte(fmt.Sprintf("%s\n\n---\n%s", string(projData), string(appData)))

	cferrors.CheckErr(delete(ctx, opts, manifests))
	awaitDeletion(ctx, opts, app)
}

func delete(ctx context.Context, opts *options, data []byte) error {
	return store.Get().NewKubeClient(ctx).Delete(ctx, &kube.DeleteOptions{
		Manifests: data,
		DryRun:    opts.dryRun,
	})
}

func awaitDeletion(ctx context.Context, opts *options, app *envman.Application) {
	awaitAppCondition(ctx, opts, app, func(a *v1alpha1.Application, err error) (bool, error) {
		if err != nil {
			if kerrors.IsGone(err) {
				return true, nil
			}

			return false, err
		}
		return false, nil
	})
}

func awaitAppCondition(ctx context.Context, opts *options, rootApp *envman.Application, predicate func(*v1alpha1.Application, error) (bool, error)) {
	o := &kube.WaitOptions{
		Interval: time.Second * 2,
		Timeout:  time.Minute * 2,
		Resources: []*kube.ResourceInfo{
			{
				Name:      rootApp.Name,
				Namespace: rootApp.Namespace,
				Func: func(ctx context.Context, c kube.Client, ns, name string) (bool, error) {
					config, err := c.ToRESTConfig()
					if err != nil {
						return false, err
					}

					argoClient, err := versioned.NewForConfig(config)
					if err != nil {
						return false, err
					}

					return predicate(argoClient.ArgoprojV1alpha1().Applications(ns).Get(ctx, name, v1.GetOptions{}))
				},
			},
		},
		DryRun: opts.dryRun,
	}

	cferrors.CheckErr(store.Get().NewKubeClient(ctx).Wait(ctx, o))
}

func cleanup(ctx context.Context) {
	log.G(ctx).Debugf("cleaning dir: %s", values.GitopsRepoClonePath)
	if err := os.RemoveAll(values.GitopsRepoClonePath); err != nil && !os.IsNotExist(err) {
		log.G(ctx).WithError(err).Error("failed to clean user local repo")
	}
}
