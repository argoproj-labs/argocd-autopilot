package install

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/codefresh-io/cf-argo/pkg/errors"
	"github.com/codefresh-io/cf-argo/pkg/git"
	"github.com/codefresh-io/cf-argo/pkg/helpers"
	"github.com/codefresh-io/cf-argo/pkg/kube"
	"github.com/codefresh-io/cf-argo/pkg/log"
	ss "github.com/codefresh-io/cf-argo/pkg/sealed-secrets"
	"github.com/codefresh-io/cf-argo/pkg/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
)

type options struct {
	repoOwner string
	repoName  string
	envName   string
	gitToken  string
	dryRun    bool
}

var values struct {
	ArgoAppsDir string
	GitToken    string
	EnvName     string
	Namespace   string
	RepoOwner   string
	RepoName    string
}

func New(ctx context.Context) *cobra.Command {
	var opts options

	cmd := &cobra.Command{
		Use:   "install",
		Short: "installs the Argo Enterprise solution on a specified cluster",
		Long:  `This command will create a new git repository that manages an Argo Enterprise solution using Argo-CD with gitops.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fillValues(&opts)
			return install(ctx, &opts)
		},
	}

	errors.MustContext(ctx, viper.BindEnv("repo-owner", "REPO_OWNER"))
	errors.MustContext(ctx, viper.BindEnv("repo-name", "REPO_NAME"))
	errors.MustContext(ctx, viper.BindEnv("env-name", "ENV_NAME"))
	errors.MustContext(ctx, viper.BindEnv("git-token", "GIT_TOKEN"))
	viper.SetDefault("repo-name", "cf-argo")

	// add kubernetes flags
	s := store.Get()
	cmd.Flags().AddFlagSet(s.KubeConfig.FlagSet(ctx))

	cmd.Flags().StringVar(&opts.repoOwner, "repo-owner", viper.GetString("repo-owner"), "name of the repository owner, defaults to [REPO_OWNER] environment variable")
	cmd.Flags().StringVar(&opts.repoName, "repo-name", viper.GetString("repo-name"), "name of the repository that will be created and used for the bootstrap installation")
	cmd.Flags().StringVar(&opts.envName, "env-name", viper.GetString("env-name"), "name of the Argo Enterprise environment to create")
	cmd.Flags().StringVar(&opts.gitToken, "git-token", viper.GetString("git-token"), "git token which will be used by argo-cd to create the gitops repository")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "when true, the command will have no side effects, and will only output the manifests to stdout")

	errors.MustContext(ctx, cmd.MarkFlagRequired("repo-owner"))
	errors.MustContext(ctx, cmd.MarkFlagRequired("env-name"))
	errors.MustContext(ctx, cmd.MarkFlagRequired("git-token"))

	return cmd
}

// fill the values used to render the templates
func fillValues(opts *options) {
	values.ArgoAppsDir = "argocd-apps"
	values.GitToken = base64.StdEncoding.EncodeToString([]byte(opts.gitToken))
	values.EnvName = opts.envName
	values.Namespace = fmt.Sprintf("%s-argocd", values.EnvName)
	values.RepoOwner = opts.repoOwner
	values.RepoName = opts.repoName
}

func install(ctx context.Context, opts *options) error {
	var err error
	defer func() {
		cleanup(ctx, err != nil, opts)
	}()

	fmt.Println("cloning template repository...")
	err = cloneBase(ctx, opts.repoName)
	if err != nil {
		return err
	}

	err = renderDir(ctx, opts.repoName)
	if err != nil {
		return err
	}

	// modify template with local data
	fmt.Println("building bootstrap resources...")
	data, err := buildBootstrapResources(ctx, opts.repoName)
	if err != nil {
		return err
	}

	if opts.dryRun {
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("applying resource to cluster...")
	err = applyBootstrapResources(ctx, data, opts)
	if err != nil {
		return err
	}

	fmt.Println("waiting for argocd initialization to complete... (might take a few seconds)")
	err = waitForDeployments(ctx)
	if err != nil {
		return err
	}

	err = createSealedSecret(ctx, opts)
	if err != nil {
		return err
	}

	err = persistGitopsRepo(ctx, opts)
	if err != nil {
		return err
	}

	err = createArgocdApp(ctx, opts)
	if err != nil {
		return err
	}

	passwd, err := getArgocdPassword(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("\n\nargocd initialized. password: %s\n", passwd)
	fmt.Printf("run: kubectl port-forward -n %s svc/argocd-server 8080:80\n\n", values.Namespace)

	return nil
}

func apply(ctx context.Context, opts *options, data []byte) error {
	d := util.DryRunNone
	if opts.dryRun {
		d = util.DryRunClient
	}
	return store.Get().NewKubeClient(ctx).Apply(ctx, &kube.ApplyOptions{
		Manifests:      data,
		DryRunStrategy: d,
	})
}

func waitForDeployments(ctx context.Context) error {
	deploymentTest := func(ctx context.Context, cs kubernetes.Interface, ns, name string) (bool, error) {
		d, err := cs.AppsV1().Deployments(ns).Get(ctx, name, v1.GetOptions{})
		if err != nil {
			return false, err
		}
		return d.Status.ReadyReplicas >= *d.Spec.Replicas, nil
	}
	ns := values.Namespace
	o := &kube.WaitOptions{
		Interval: time.Second * 2,
		Timeout:  time.Minute * 2,
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
	}

	return store.Get().NewKubeClient(ctx).Wait(ctx, o)
}

func getArgocdPassword(ctx context.Context) (string, error) {
	cs, err := store.Get().NewKubeClient(ctx).KubernetesClientSet()
	if err != nil {
		return "", err
	}
	secret, err := cs.CoreV1().Secrets(values.Namespace).Get(ctx, "argocd-initial-admin-secret", v1.GetOptions{})
	if err != nil {
		return "", err
	}
	passwd, ok := secret.Data["password"]
	if !ok {
		return "", fmt.Errorf("argocd initial password not found")
	}

	return string(passwd), nil
}

func createArgocdApp(ctx context.Context, opts *options) error {
	data, err := ioutil.ReadFile(filepath.Join(
		values.RepoName,
		values.ArgoAppsDir,
		fmt.Sprintf("%s.yaml", values.EnvName),
	))
	if err != nil {
		return err
	}

	return apply(ctx, opts, data)
}

func createSealedSecret(ctx context.Context, opts *options) error {
	s, err := ss.CreateSealedSecretFromSecretFile(ctx, values.Namespace, filepath.Join(values.RepoName, "secret.yaml"))
	if err != nil {
		return err
	}

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	err = apply(ctx, opts, data)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(
		filepath.Join(
			values.RepoName,
			"kustomize",
			"components",
			"argo-cd",
			"overlays",
			values.EnvName,
			"sealed-secret.json",
		),
		data,
		0644,
	)
}

func applyBootstrapResources(ctx context.Context, manifests []byte, opts *options) error {
	d := util.DryRunNone
	if opts.dryRun {
		d = util.DryRunClient
	}
	return store.Get().NewKubeClient(ctx).Apply(ctx, &kube.ApplyOptions{
		Manifests:      manifests,
		DryRunStrategy: d,
	})
}

func renderDir(ctx context.Context, path string) error {
	if err := helpers.RenameEnvNameRecurse(ctx, path, values.EnvName); err != nil {
		return err
	}

	return helpers.RenderDirRecurse(filepath.Join(path, "**/*.*"), values)
}

func buildBootstrapResources(ctx context.Context, path string) ([]byte, error) {
	opts := krusty.MakeDefaultOptions()
	opts.DoLegacyResourceSort = true

	k := krusty.MakeKustomizer(filesys.MakeFsOnDisk(), opts)
	res, err := k.Run(path)
	if err != nil {
		return nil, err
	}

	data, err := res.AsYaml()
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("").Parse(string(data))
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(make([]byte, 0, 4096))

	err = tpl.Execute(buf, values)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func persistGitopsRepo(ctx context.Context, opts *options) error {
	r, err := git.Init(ctx, opts.repoName)
	if err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(opts.repoName, "*.yaml"))
	if err != nil {
		return err
	}
	for _, f := range files {
		log.G(ctx).WithField("path", f).Debug("removing file")
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	err = r.Add(ctx, ".")
	if err != nil {
		return err
	}

	_, err = r.Commit(ctx, "Initial commit")
	if err != nil {
		return err
	}

	fmt.Printf("creating gitops repository: %s/%s...\n", opts.repoOwner, opts.repoName)
	cloneURL, err := createRemoteRepo(ctx, opts)
	if err != nil {
		return err
	}

	err = r.AddRemote(ctx, "origin", cloneURL)
	if err != nil {
		return err
	}

	fmt.Println("pushing to gitops repo...")
	err = r.Push(ctx, &git.PushOptions{
		Auth: &git.Auth{
			Password: opts.gitToken,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func createRemoteRepo(ctx context.Context, opts *options) (string, error) {
	p, err := git.New(&git.Options{
		Type: "github", // TODO: support other types
		Auth: &git.Auth{
			Password: opts.gitToken,
		},
	})
	if err != nil {
		return "", err
	}

	cloneURL, err := p.CreateRepository(ctx, &git.CreateRepositoryOptions{
		Owner:   opts.repoOwner,
		Name:    opts.repoName,
		Private: true,
	})
	if err != nil {
		return "", err
	}

	return cloneURL, err
}

func cloneBase(ctx context.Context, path string) error {
	baseGitURL := store.Get().BaseGitURL
	_, err := git.Clone(ctx, &git.CloneOptions{
		URL:  baseGitURL,
		Path: path,
	})
	if err != nil {
		return err
	}

	err = os.RemoveAll(filepath.Join(path, ".git"))
	if err != nil {
		return err
	}

	return err
}

func cleanup(ctx context.Context, failed bool, opts *options) {
	if failed {
		log.G(ctx).Debugf("cleaning local user repo: %s", opts.repoName)
		if err := os.RemoveAll(opts.repoName); err != nil && !os.IsNotExist(err) {
			log.G(ctx).WithError(err).Error("failed to clean user local repo")
		}
	}
}
