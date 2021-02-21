package install

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	envman "github.com/codefresh-io/cf-argo/pkg/environments-manager"
	cferrors "github.com/codefresh-io/cf-argo/pkg/errors"
	"github.com/codefresh-io/cf-argo/pkg/git"
	"github.com/codefresh-io/cf-argo/pkg/helpers"
	"github.com/codefresh-io/cf-argo/pkg/kube"
	"github.com/codefresh-io/cf-argo/pkg/log"
	ss "github.com/codefresh-io/cf-argo/pkg/sealed-secrets"
	"github.com/codefresh-io/cf-argo/pkg/store"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type options struct {
	repoURL  string
	envName  string
	gitToken string
	baseRepo string
	dryRun   bool
}

var values struct {
	BootstrapDir          string
	Namespace             string
	RepoOwner             string
	RepoName              string
	TemplateRepoClonePath string
	GitopsRepoClonePath   string
	GitopsRepo            git.Repository
}

var renderValues struct {
	EnvName   string
	RepoOwner string
	RepoName  string
	GitToken  string
}

func New(ctx context.Context) *cobra.Command {
	var opts options

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Installs the Argo Enterprise solution on a specified cluster",
		Long:  "This command will create a new git repository that manages an Argo Enterprise solution using Argo-CD with gitops.",
		Run: func(cmd *cobra.Command, args []string) {
			fillValues(&opts)
			install(ctx, &opts)
		},
	}

	// add kubernetes flags
	store.Get().KubeConfig.AddFlagSet(cmd)

	_ = viper.BindEnv("repo-url", "REPO_URL")
	_ = viper.BindEnv("env-name", "ENV_NAME")
	_ = viper.BindEnv("git-token", "GIT_TOKEN")
	_ = viper.BindEnv("base-repo", "BASE_REPO")
	viper.SetDefault("env-name", "production")
	viper.SetDefault("base-repo", store.Get().BaseGitURL)
	viper.SetDefault("dry-run", false)

	cmd.Flags().StringVar(&opts.repoURL, "repo-url", viper.GetString("repo-url"), "the gitops repository url. If it does not exist we will try to create it for you [REPO_URL]")
	cmd.Flags().StringVar(&opts.envName, "env-name", viper.GetString("env-name"), "name of the Argo Enterprise environment to create")
	cmd.Flags().StringVar(&opts.gitToken, "git-token", viper.GetString("git-token"), "git token which will be used by argo-cd to create the gitops repository")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", viper.GetBool("dry-run"), "when true, the command will have no side effects, and will only output the manifests to stdout")
	cmd.Flags().StringVar(&opts.baseRepo, "base-repo", viper.GetString("base-repo"), "the template repository url")

	cferrors.MustContext(ctx, cmd.MarkFlagRequired("repo-url"))
	cferrors.MustContext(ctx, cmd.MarkFlagRequired("git-token"))
	cferrors.MustContext(ctx, cmd.Flags().MarkHidden("base-repo"))

	return cmd
}

// fill the values used to render the templates
func fillValues(opts *options) {
	var err error
	values.RepoOwner, values.RepoName, err = git.SplitCloneURL(opts.repoURL)
	cferrors.CheckErr(err)

	values.BootstrapDir = "bootstrap"
	values.Namespace = fmt.Sprintf("%s-argocd", opts.envName)

	renderValues.EnvName = opts.envName
	renderValues.RepoOwner = values.RepoOwner
	renderValues.RepoName = values.RepoName
	renderValues.GitToken = base64.StdEncoding.EncodeToString([]byte(opts.gitToken))
}

func install(ctx context.Context, opts *options) {
	defer func() {
		cleanup(ctx)
		if err := recover(); err != nil {
			panic(err)
		}
	}()

	log.G(ctx).Printf("cloning template repository...")
	conf := tryCloneExistingRepo(ctx, opts)

	prepareBase(ctx, opts)

	if conf == nil {
		initializeNewGitopsRepo(ctx, opts)
	} else {
		addToExistingGitopsRepo(ctx, conf, opts)
	}

	log.G(ctx).Printf("waiting for argocd initialization to complete... (might take a few seconds)")
	waitForDeployments(ctx, opts)

	createSealedSecret(ctx, opts)

	persistGitopsRepo(ctx, opts)

	createArgocdApp(ctx, opts)

	printArgocdData(ctx, opts)
}

func prepareBase(ctx context.Context, opts *options) {
	var err error
	log.G(ctx).Debug("creating temp dir for template repo")
	values.TemplateRepoClonePath, err = ioutil.TempDir("", "tpl-")
	cferrors.CheckErr(err)

	log.G(ctx).WithField("location", values.TemplateRepoClonePath).Debug("temp dir created")

	_, err = git.Clone(ctx, &git.CloneOptions{
		URL:  opts.baseRepo,
		Path: values.TemplateRepoClonePath,
	})
	cferrors.CheckErr(err)

	log.G(ctx).Debug("cleaning template repository")
	cferrors.CheckErr(os.RemoveAll(filepath.Join(values.TemplateRepoClonePath, ".git")))

	log.G(ctx).Debug("renaming envName files")
	cferrors.CheckErr(helpers.RenameFilesWithEnvName(ctx, values.TemplateRepoClonePath, opts.envName))

	log.G(ctx).Debug("rendering template values")
	cferrors.CheckErr(helpers.RenderDirRecurse(filepath.Join(values.TemplateRepoClonePath, "**/*.*"), renderValues))
}

func tryCloneExistingRepo(ctx context.Context, opts *options) *envman.Config {
	p, err := git.NewProvider(&git.Options{
		Type: "github", // only option for now
		Auth: &git.Auth{
			Password: opts.gitToken,
		},
	})
	cferrors.CheckErr(err)

	values.GitopsRepo, err = p.CloneRepository(ctx, &git.GetRepositoryOptions{
		Owner: values.RepoOwner,
		Name:  values.RepoName,
	})

	if err != nil {
		if err != git.ErrRepoNotFound {
			cferrors.CheckErr(err)
		}

		return nil // we will create it later
	}

	values.GitopsRepoClonePath, err = values.GitopsRepo.Root()
	cferrors.CheckErr(err)

	conf, err := envman.LoadConfig(values.GitopsRepoClonePath)
	cferrors.CheckErr(err)

	if _, exists := conf.Environments[opts.envName]; exists {
		panic(fmt.Errorf("environment with name \"%s\" already exists in target repository", opts.envName))
	}

	return conf
}

func waitForDeployments(ctx context.Context, opts *options) {
	deploymentTest := func(ctx context.Context, c kube.Client, ns, name string) (bool, error) {
		cs, err := c.KubernetesClientSet()
		if err != nil {
			return false, err
		}

		d, err := cs.AppsV1().Deployments(ns).Get(ctx, name, v1.GetOptions{})
		if err != nil {
			return false, err
		}

		return d.Status.ReadyReplicas >= *d.Spec.Replicas, nil
	}
	ns := values.Namespace
	o := &kube.WaitOptions{
		Interval: time.Second * 2,
		Timeout:  time.Minute * 5,
		Resources: []*kube.ResourceInfo{
			{
				Name:      "argocd-server",
				Namespace: ns,
				Func:      deploymentTest,
			},
			{
				Name:      "sealed-secrets-controller",
				Namespace: ns,
				Func:      deploymentTest,
			},
		},
		DryRun: opts.dryRun,
	}

	cferrors.CheckErr(store.Get().NewKubeClient(ctx).Wait(ctx, o))
}

func printArgocdData(ctx context.Context, opts *options) {
	if opts.dryRun {
		return
	}

	cs, err := store.Get().NewKubeClient(ctx).KubernetesClientSet()
	cferrors.CheckErr(err)

	secret, err := cs.CoreV1().Secrets(values.Namespace).Get(ctx, "argocd-initial-admin-secret", v1.GetOptions{})
	cferrors.CheckErr(err)
	passwd, ok := secret.Data["password"]
	if !ok {
		panic(fmt.Errorf("argocd initial password not found"))
	}

	log.G(ctx).Printf("\n\nargocd initialized. password: %s", passwd)
	log.G(ctx).Printf("run: kubectl port-forward -n %s svc/argocd-server 8080:80", values.Namespace)
}

func createArgocdApp(ctx context.Context, opts *options) {
	tplConf, err := envman.LoadConfig(values.GitopsRepoClonePath)
	cferrors.CheckErr(err)
	absArgoAppsDir := filepath.Join(values.GitopsRepoClonePath, filepath.Dir(tplConf.Environments[opts.envName].RootApplicationPath))

	projData, err := ioutil.ReadFile(filepath.Join(absArgoAppsDir, fmt.Sprintf("%s-project.yaml", opts.envName)))
	cferrors.CheckErr(err)

	appData, err := ioutil.ReadFile(filepath.Join(absArgoAppsDir, fmt.Sprintf("%s.yaml", opts.envName)))
	cferrors.CheckErr(err)

	manifests := []byte(fmt.Sprintf("%s\n\n---\n%s", string(projData), string(appData)))

	cferrors.CheckErr(apply(ctx, opts, manifests))
}

func createSealedSecret(ctx context.Context, opts *options) {
	secretPath := filepath.Join(values.TemplateRepoClonePath, values.BootstrapDir, "secret.yaml")
	s, err := ss.CreateSealedSecretFromSecretFile(ctx, values.Namespace, secretPath, opts.dryRun)
	cferrors.CheckErr(err)

	data, err := json.Marshal(s)
	cferrors.CheckErr(err)

	err = apply(ctx, opts, data)
	cferrors.CheckErr(err)

	cferrors.CheckErr(ioutil.WriteFile(
		filepath.Join(
			values.TemplateRepoClonePath,
			"kustomize",
			"components",
			"argo-cd",
			"overlays",
			opts.envName,
			"sealed-secret.json",
		),
		data,
		0644,
	))
}

func persistGitopsRepo(ctx context.Context, opts *options) {
	if opts.dryRun {
		return
	}

	isNewRepo, err := values.GitopsRepo.IsNewRepo()
	cferrors.CheckErr(err)

	if isNewRepo {
		log.G(ctx).Printf("creating gitops repository: %s/%s...", values.RepoOwner, values.RepoName)
		cloneURL, err := createRemoteRepo(ctx, opts)
		cferrors.CheckErr(err)

		cferrors.CheckErr(values.GitopsRepo.AddRemote(ctx, "origin", cloneURL))
	}

	cferrors.CheckErr(os.RemoveAll(filepath.Join(values.TemplateRepoClonePath, values.BootstrapDir)))

	cferrors.CheckErr(values.GitopsRepo.Add(ctx, "."))

	_, err = values.GitopsRepo.Commit(ctx, fmt.Sprintf("added environment %s", opts.envName))
	cferrors.CheckErr(err)

	log.G(ctx).Printf("pushing to gitops repo...")
	err = values.GitopsRepo.Push(ctx, &git.PushOptions{
		Auth: &git.Auth{
			Password: opts.gitToken,
		},
	})
	cferrors.CheckErr(err)
}

func initializeNewGitopsRepo(ctx context.Context, opts *options) {
	var err error
	// use the template repo to init the new repo
	values.GitopsRepoClonePath = values.TemplateRepoClonePath
	values.GitopsRepo, err = git.Init(ctx, values.GitopsRepoClonePath)
	cferrors.CheckErr(err)

	conf, err := envman.LoadConfig(values.GitopsRepoClonePath)
	cferrors.CheckErr(err)

	env := conf.FirstEnv()
	env.UpdateTemplateRef(opts.baseRepo)
	cferrors.CheckErr(conf.Persist())

	log.G(ctx).Printf("installing bootstrap resources...")
	cferrors.CheckErr(env.ApplyBootstrap(ctx, renderValues, opts.dryRun))
}

func addToExistingGitopsRepo(ctx context.Context, conf *envman.Config, opts *options) {
	tplConf, err := envman.LoadConfig(values.TemplateRepoClonePath)
	cferrors.CheckErr(err)

	newEnv := tplConf.FirstEnv()
	newEnv.UpdateTemplateRef(opts.baseRepo)
	cferrors.CheckErr(conf.AddEnvironmentP(newEnv))

	log.G(ctx).Printf("installing bootstrap resources...")
	cferrors.CheckErr(newEnv.ApplyBootstrap(ctx, renderValues, opts.dryRun))
}

func apply(ctx context.Context, opts *options, data []byte) error {
	return store.Get().NewKubeClient(ctx).Apply(ctx, &kube.ApplyOptions{
		Manifests: data,
		DryRun:    opts.dryRun,
	})
}

func createRemoteRepo(ctx context.Context, opts *options) (string, error) {
	p, err := git.NewProvider(&git.Options{
		Type: "github", // need to support other types
		Auth: &git.Auth{
			Password: opts.gitToken,
		},
	})
	if err != nil {
		return "", err
	}

	cloneURL, err := p.CreateRepository(ctx, &git.CreateRepositoryOptions{
		Owner:   values.RepoOwner,
		Name:    values.RepoName,
		Private: true,
	})
	if err != nil {
		return "", err
	}

	return cloneURL, err
}

func cleanup(ctx context.Context) {
	log.G(ctx).Debugf("cleaning dirs: %s", strings.Join([]string{values.GitopsRepoClonePath, values.TemplateRepoClonePath}, ","))
	if err := os.RemoveAll(values.GitopsRepoClonePath); err != nil && !os.IsNotExist(err) {
		log.G(ctx).WithError(err).Error("failed to clean user local repo")
	}
	if err := os.RemoveAll(values.TemplateRepoClonePath); err != nil && !os.IsNotExist(err) {
		log.G(ctx).WithError(err).Error("failed to clean template repo")
	}
}
