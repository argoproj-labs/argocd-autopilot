package application

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/ghodss/yaml"

	argocdutil "github.com/argoproj/argo-cd/v2/cmd/util"
	argocdapp "github.com/argoproj/argo-cd/v2/pkg/apis/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdsettings "github.com/argoproj/argo-cd/v2/util/settings"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
	kusttypes "sigs.k8s.io/kustomize/api/types"
)

type Application interface {
	GenerateManifests() ([]byte, error)
	ArgoCD() *v1alpha1.Application
}

type CreateOptions struct {
	AppSpecifier   string
	AppName        string
	SrcPath        string
	argoAppOptions argocdutil.AppOptions
	flags          *pflag.FlagSet
}

type application struct {
	tag       string
	name      string
	namespace string
	path      string
	fs        filesys.FileSystem
	argoApp   *v1alpha1.Application
}

type bootstrapApp struct {
	*application
	repoUrl string
}

func AddFlags(cmd *cobra.Command, defAppName string) *CreateOptions {
	co := &CreateOptions{}

	cmd.Flags().StringVar(&co.AppSpecifier, "app", "", "The application specifier")
	cmd.Flags().StringVar(&co.AppName, "app-name", defAppName, "The application name")

	argocdutil.AddAppFlags(cmd, &co.argoAppOptions)

	co.flags = cmd.Flags()

	return co
}

func (app *application) GenerateManifests() ([]byte, error) {
	return app.kustomizeBuild() // TODO: supporting only kustomize for now
}

func (o *CreateOptions) Parse(bootstrap bool) (Application, error) {
	if o.AppSpecifier == "" {
		return nil, fmt.Errorf("empty app specifier not allowed")
	}

	namespace, err := o.flags.GetString("namespace")
	if err != nil {
		return nil, err
	}

	argoApp, err := argocdutil.ConstructApp("", o.AppName, getLabels(o.AppName), []string{}, o.argoAppOptions, o.flags)
	if err != nil {
		return nil, err
	}

	// set default options
	argoApp.Spec.SyncPolicy = &v1alpha1.SyncPolicy{
		Automated: &v1alpha1.SyncPolicyAutomated{
			SelfHeal: true,
			Prune:    true,
		},
	}

	app := &application{
		path:      o.AppSpecifier, // TODO: supporting only path for now
		namespace: namespace,
		fs:        filesys.MakeFsOnDisk(),
		argoApp:   argoApp,
	}

	if bootstrap {
		app.argoApp.ObjectMeta.Namespace = namespace // override "default" namespace
		app.argoApp.Spec.Destination.Server = "https://kubernetes.default.svc"
		app.argoApp.Spec.Destination.Namespace = namespace
		app.argoApp.Spec.Source.Path = o.SrcPath

		return &bootstrapApp{
			application: app,
			repoUrl:     util.MustGetString(o.flags, "repo"),
		}, nil
	}

	return app, nil
}

func (o *CreateOptions) ParseOrDie(bootstrap bool) Application {
	app, err := o.Parse(bootstrap)
	util.Die(err)
	return app
}

func (app *application) kustomizeBuild() ([]byte, error) {
	kopts := krusty.MakeDefaultOptions()
	kopts.DoLegacyResourceSort = true

	k := krusty.MakeKustomizer(kopts)

	log.G().WithField("path", app.path).Debug("running kustomize")
	res, err := k.Run(app.fs, app.path)
	if err != nil {
		return nil, err
	}

	return res.AsYaml()
}

func (app *application) ArgoCD() *v1alpha1.Application {
	return app.argoApp
}

func (app *bootstrapApp) GenerateManifests() ([]byte, error) {
	opts := krusty.MakeDefaultOptions()
	opts.DoLegacyResourceSort = true
	kust := krusty.MakeKustomizer(opts)
	fs := filesys.MakeFsOnDisk()

	kustPath, resourcePath, err := getBootstrapPaths(app.path)
	if err != nil {
		return nil, err
	}
	kustPathDir := filepath.Dir(kustPath)
	defer os.RemoveAll(kustPathDir)

	kyaml, err := createBootstrapKustomization(resourcePath, app.repoUrl, app.namespace)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(kustPath, kyaml, 0400)
	if err != nil {
		return nil, err
	}

	log.G().WithFields(log.Fields{
		"bootstrapKustPath": kustPath,
		"resourcePath":      resourcePath,
	}).Debugf("running bootstrap kustomization: %s\n", string(kyaml))

	res, err := kust.Run(fs, kustPathDir)
	if err != nil {
		return nil, err
	}

	bootstrapManifests, err := res.AsYaml()
	if err != nil {
		return nil, err
	}

	return util.JoinManifests(createNamespace(app.namespace), bootstrapManifests), nil
}

func getLabels(appName string) []string {
	return []string{
		"app.kubernetes.io/managed-by=argo-autopilot",
		"app.kubernetes.io/name=" + appName,
	}
}

func createNamespace(namespace string) []byte {
	ns := &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	data, err := yaml.Marshal(ns)
	util.Die(err)

	return data
}

func createCreds(repoUrl string) ([]byte, error) {
	creds := []argocdsettings.Repository{
		{
			URL: repoUrl,
			UsernameSecret: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "autopilot-secret",
				},
				Key: "git_username",
			},
			PasswordSecret: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "autopilot-secret",
				},
				Key: "git_token",
			},
		},
	}

	return yaml.Marshal(creds)
}

func createBootstrapKustomization(resourcePath, repoURL, namespace string) ([]byte, error) {
	credsYAML, err := createCreds(repoURL)
	if err != nil {
		return nil, err
	}

	k := &kusttypes.Kustomization{
		Resources: []string{resourcePath},
		TypeMeta: kusttypes.TypeMeta{
			APIVersion: kusttypes.KustomizationVersion,
			Kind:       kusttypes.KustomizationKind,
		},
		ConfigMapGenerator: []kusttypes.ConfigMapArgs{
			{
				GeneratorArgs: kusttypes.GeneratorArgs{
					Name:     "argocd-cm",
					Behavior: kusttypes.BehaviorMerge.String(),
					KvPairSources: kusttypes.KvPairSources{
						LiteralSources: []string{
							"repository.credentials=" + string(credsYAML),
						},
					},
				},
			},
		},
		Namespace: namespace,
	}

	k.FixKustomizationPostUnmarshalling()
	errs := k.EnforceFields()
	if len(errs) > 0 {
		return nil, fmt.Errorf("kustomization errors: %s", strings.Join(errs, "\n"))
	}
	k.FixKustomizationPreMarshalling()

	return yaml.Marshal(k)
}

func getBootstrapPaths(path string) (string, string, error) {
	var err error
	td, err := ioutil.TempDir("", "auto-pilot")
	if err != nil {
		return "", "", err
	}

	kustPath := filepath.Join(td, "kustomization.yaml")

	// for example: github.com/codefresh-io/argocd-autopilot/manifests
	if _, err = os.Stat(path); err != nil && os.IsNotExist(err) {
		return kustPath, path, nil
	}

	// local file (in the filesystem)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}

	resourcePath, err := filepath.Rel(kustPath, absPath)
	if err != nil {
		return "", "", err
	}

	return kustPath, resourcePath, nil
}

func NewRootApp(namespace, repoURL, srcPath, revision string) Application {
	return &application{
		argoApp: &v1alpha1.Application{
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
					RepoURL:        repoURL,
					Path:           srcPath,
					TargetRevision: revision,
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
		},
	}
}
