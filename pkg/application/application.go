package application

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
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

const (
	defaultDestServer = "https://kubernetes.default.svc"
)

type Application interface {
	// GenerateManifests runs kustomize build on the app and returns the result.
	GenerateManifests() ([]byte, error)

	// ArgoCD parses the app flags and returns the constructed argo-cd application.
	ArgoCD() *v1alpha1.Application

	// Kustomization returns the marshaled kustomization file for the bootstrap
	// application. only available when creating bootstrap application.
	Kustomization() ([]byte, error)

	// Base returns the base kustomization file for this app.
	Base() *kusttypes.Kustomization

	// Overlay returns the overlay kustomization object that is looking on this
	// app.Base() file.
	Overlay() *kusttypes.Kustomization

	// Config returns this app's config.json file that should be next to the overlay
	// kustomization.yaml file. This is used by the environment's application set
	// to generate the final argo-cd application.
	ConfigJson() *Config
}

type Config struct {
	AppName       string `json:"appName,omitempty"`
	UserGivenName string `json:"userGivenName,omitempty"`
	DestNamespace string `json:"destNamespace,omitempty"`
	DestServer    string `json:"destServer,omitempty"`
}

type CreateOptions struct {
	AppSpecifier   string
	AppName        string
	SrcPath        string
	Namespace      string
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

/*********************************/
/*       CreateOptions impl      */
/*********************************/
func (o *CreateOptions) Parse(bootstrap bool) (Application, error) {
	if o.AppSpecifier == "" {
		return nil, fmt.Errorf("empty app specifier not allowed")
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
		namespace: o.Namespace,
		fs:        filesys.MakeFsOnDisk(),
		argoApp:   argoApp,
	}

	if bootstrap {
		app.argoApp.ObjectMeta.Namespace = app.namespace
		app.argoApp.Spec.Destination.Server = defaultDestServer
		app.argoApp.Spec.Destination.Namespace = app.namespace
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

/*********************************/
/*        Application impl       */
/*********************************/

func (app *application) GenerateManifests() ([]byte, error) {
	return app.kustomizeBuild() // TODO: supporting only kustomize for now
}

func (app *application) ArgoCD() *v1alpha1.Application {
	return app.argoApp
}

func (app *application) Kustomization() ([]byte, error) {
	return nil, nil
}

func (app *application) Overlay() *kusttypes.Kustomization {
	return &kusttypes.Kustomization{
		Resources: []string{"../base"},
		TypeMeta: kusttypes.TypeMeta{
			APIVersion: kusttypes.KustomizationVersion,
			Kind:       kusttypes.KustomizationKind,
		},
	}
}

func (app *application) Base() *kusttypes.Kustomization {
	return &kusttypes.Kustomization{
		Resources: []string{app.path},
		TypeMeta: kusttypes.TypeMeta{
			APIVersion: kusttypes.KustomizationVersion,
			Kind:       kusttypes.KustomizationKind,
		},
	}
}

func (app *application) ConfigJson() *Config {
	return &Config{
		AppName:       app.name,
		UserGivenName: app.ArgoCD().Name,
		DestNamespace: app.ArgoCD().Spec.Destination.Namespace,
		DestServer:    app.ArgoCD().Spec.Destination.Server,
	}
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

/*********************************/
/*   Bootstrap application impl  */
/*********************************/

func (app *bootstrapApp) GenerateManifests() ([]byte, error) {
	td, err := ioutil.TempDir("", "auto-pilot")
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(td)
	kustPath := filepath.Join(td, "kustomization.yaml")
	kyaml, err := app.Kustomization()
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(kustPath, kyaml, 0400)
	if err != nil {
		return nil, err
	}

	log.G().WithFields(log.Fields{
		"bootstrapKustPath": kustPath,
		"resourcePath":      app.path,
	}).Debugf("running bootstrap kustomization: %s\n", string(kyaml))

	opts := krusty.MakeDefaultOptions()
	opts.DoLegacyResourceSort = true
	kust := krusty.MakeKustomizer(opts)
	fs := filesys.MakeFsOnDisk()
	res, err := kust.Run(fs, filepath.Dir(kustPath))
	if err != nil {
		return nil, err
	}

	bootstrapManifests, err := res.AsYaml()
	if err != nil {
		return nil, err
	}

	return util.JoinManifests(createNamespace(app.namespace), bootstrapManifests), nil
}

func (app *bootstrapApp) Kustomization() ([]byte, error) {
	credsYAML, err := createCreds(app.repoUrl)
	if err != nil {
		return nil, err
	}

	k := &kusttypes.Kustomization{
		Resources: []string{app.path},
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
		Namespace: app.namespace,
	}

	k.FixKustomizationPostUnmarshalling()
	errs := k.EnforceFields()
	if len(errs) > 0 {
		return nil, fmt.Errorf("kustomization errors: %s", strings.Join(errs, "\n"))
	}
	k.FixKustomizationPreMarshalling()

	return yaml.Marshal(k)
}

func (app *bootstrapApp) Overlay() *kusttypes.Kustomization {
	return nil
}

func (app *bootstrapApp) Base() *kusttypes.Kustomization {
	return nil
}

func (app *bootstrapApp) ConfigJson() *Config {
	return nil
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
					"app.kubernetes.io/managed-by": store.Common.ManagedBy,
					"app.kubernetes.io/name":       store.Common.RootName,
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
