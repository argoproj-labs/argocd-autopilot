package application

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	appset "github.com/argoproj-labs/applicationset/api/v1alpha1"
	appsetv1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	argocdutil "github.com/argoproj/argo-cd/v2/cmd/util"
	argocdapp "github.com/argoproj/argo-cd/v2/pkg/apis/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdsettings "github.com/argoproj/argo-cd/v2/util/settings"
	"github.com/ghodss/yaml"
	"github.com/go-git/go-billy/v5"
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

var (
	// Errors
	ErrEmptyAppSpecifier = errors.New("empty app specifier not allowed")
	ErrEmptyAppName      = errors.New("app name cannot be empty, please specify application name with: --app-name")
)

type (
	Application interface {
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

	BootstrapApplication interface {
		// GenerateManifests runs kustomize build on the app and returns the result.
		GenerateManifests() ([]byte, error)

		// Kustomization returns the kustomization for the bootstrap application.
		//  only available when creating bootstrap application.
		Kustomization() (*kusttypes.Kustomization, error)

		// CreateApp returns an argocd application that watches the gitops
		// repo at the specified path and revision
		CreateApp(name, revision, srcPath string) *v1alpha1.Application
	}

	Config struct {
		AppName       string `json:"appName,omitempty"`
		UserGivenName string `json:"userGivenName,omitempty"`
		DestNamespace string `json:"destNamespace,omitempty"`
		DestServer    string `json:"destServer,omitempty"`
	}

	CreateOptions struct {
		AppSpecifier   string
		AppName        string
		SrcPath        string
		Namespace      string
		Server         string
		argoAppOptions argocdutil.AppOptions
		flags          *pflag.FlagSet
	}

	GenerateAppSetOptions struct {
		FS        billy.Filesystem
		Name      string
		Namespace string
		RepoURL   string
		Revision  string
	}

	application struct {
		tag       string
		name      string
		namespace string
		path      string
		kustPath  string
		fs        filesys.FileSystem
		argoApp   *v1alpha1.Application
	}

	bootstrapApp struct {
		*application
		repoUrl string
	}
)

func AddFlags(cmd *cobra.Command, defAppName string) *CreateOptions {
	co := &CreateOptions{}

	cmd.Flags().StringVar(&co.AppSpecifier, "app", "", "The application specifier (e.g. argocd@v1.0.2 | https://github.com")
	cmd.Flags().StringVar(&co.AppName, "app-name", defAppName, "The application name")
	cmd.Flags().StringVar(&co.Server, "dest-server", "", fmt.Sprintf("K8s cluster URL (e.g. %s)", defaultDestServer))
	cmd.Flags().StringVar(&co.Namespace, "dest-namespace", "", "K8s target namespace (overrides the namespace specified in the ksonnet app.yaml)")

	co.flags = cmd.Flags()

	return co
}

/*********************************/
/*       CreateOptions impl      */
/*********************************/
func (o *CreateOptions) Parse() (Application, error) {
	return parseApplication(o)
}

func (o *CreateOptions) ParseBootstrap() (BootstrapApplication, error) {
	app, err := parseApplication(o)
	if err != nil {
		return nil, err
	}

	return &bootstrapApp{
		application: app,
		repoUrl:     o.flags.Lookup("repo").Value.String(),
	}, nil
}

func GenerateApplicationSet(o *GenerateAppSetOptions) *appset.ApplicationSet {
	return generateAppSet(o)
}

/*********************************/
/*        Application impl       */
/*********************************/
func (app *application) Base() *kusttypes.Kustomization {
	return &kusttypes.Kustomization{
		Resources: []string{app.path},
		TypeMeta: kusttypes.TypeMeta{
			APIVersion: kusttypes.KustomizationVersion,
			Kind:       kusttypes.KustomizationKind,
		},
	}
}

func (app *application) Overlay() *kusttypes.Kustomization {
	return &kusttypes.Kustomization{
		Resources: []string{"../../base"},
		TypeMeta: kusttypes.TypeMeta{
			APIVersion: kusttypes.KustomizationVersion,
			Kind:       kusttypes.KustomizationKind,
		},
	}
}

func (app *application) ConfigJson() *Config {
	return &Config{
		AppName:       app.argoApp.Name,
		UserGivenName: app.argoApp.Name,
		DestNamespace: app.argoApp.Spec.Destination.Namespace,
		DestServer:    app.argoApp.Spec.Destination.Server,
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
	k, err := app.Kustomization()
	if err != nil {
		return nil, err
	}

	kyaml, err := yaml.Marshal(k)
	if err != nil {
		return nil, err
	}

	if err = ioutil.WriteFile(app.kustPath, kyaml, 0400); err != nil {
		return nil, err
	}

	log.G().WithFields(log.Fields{
		"bootstrapKustPath": app.kustPath,
		"resourcePath":      app.path,
	}).Debugf("running bootstrap kustomization: %s\n", string(kyaml))

	opts := krusty.MakeDefaultOptions()
	opts.DoLegacyResourceSort = true
	kust := krusty.MakeKustomizer(opts)
	fs := filesys.MakeFsOnDisk()
	res, err := kust.Run(fs, filepath.Dir(app.kustPath))
	if err != nil {
		return nil, err
	}

	bootstrapManifests, err := res.AsYaml()
	if err != nil {
		return nil, err
	}

	return util.JoinManifests(createNamespace(app.namespace), bootstrapManifests), nil
}

func (app *bootstrapApp) Kustomization() (*kusttypes.Kustomization, error) {
	credsYAML, err := createCreds(app.repoUrl)
	if err != nil {
		return nil, err
	}

	td, err := ioutil.TempDir("", "auto-pilot")
	if err != nil {
		return nil, err
	}

	app.kustPath = filepath.Join(td, "kustomization.yaml")

	srcPath, err := getBootstrapPaths(app.path, app.kustPath)

	k := &kusttypes.Kustomization{
		Resources: []string{srcPath},
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

	return k, nil
}

func (app *bootstrapApp) CreateApp(name, revision, srcPath string) *v1alpha1.Application {
	return &v1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: argocdapp.Group + "/v1alpha1",
			Kind:       argocdapp.ApplicationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: app.namespace,
			Name:      name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": store.Default.ManagedBy,
				"app.kubernetes.io/name":       name,
			},
			Finalizers: []string{
				"resources-finalizer.argocd.argoproj.io",
			},
		},
		Spec: v1alpha1.ApplicationSpec{
			Project: "default",
			Source: v1alpha1.ApplicationSource{
				RepoURL:        app.repoUrl,
				Path:           srcPath,
				TargetRevision: revision,
			},
			Destination: v1alpha1.ApplicationDestination{
				Server:    defaultDestServer,
				Namespace: app.namespace,
			},
			SyncPolicy: &v1alpha1.SyncPolicy{
				Automated: &v1alpha1.SyncPolicyAutomated{
					SelfHeal: true,
					Prune:    true,
				},
			},
			IgnoreDifferences: []v1alpha1.ResourceIgnoreDifferences{
				{
					Group: "argoproj.io",
					Kind:  "Application",
					JSONPointers: []string{
						"/status",
					},
				},
			},
		},
	}
}

func parseApplication(o *CreateOptions) (*application, error) {
	if o.AppSpecifier == "" {
		return nil, ErrEmptyAppSpecifier
	}

	if o.AppName == "" {
		return nil, ErrEmptyAppName
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
		name:      o.AppName,
		path:      o.AppSpecifier, // TODO: supporting only path for now
		namespace: o.Namespace,
		fs:        filesys.MakeFsOnDisk(),
		argoApp:   argoApp,
	}

	return app, nil
}

func generateAppSet(o *GenerateAppSetOptions) *appset.ApplicationSet {
	return &appset.ApplicationSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApplicationSet",
			APIVersion: appset.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.Name,
			Namespace: o.Namespace,
		},
		Spec: appset.ApplicationSetSpec{
			Generators: []appset.ApplicationSetGenerator{
				{
					Git: &appset.GitGenerator{
						RepoURL:  o.RepoURL,
						Revision: o.Revision,
						Files: []appset.GitFileGeneratorItem{
							{
								Path: o.FS.Join("kustomize", "**", "overlays", o.Name, "config.json"),
							},
						},
					},
				},
			},
			Template: appset.ApplicationSetTemplate{
				ApplicationSetTemplateMeta: appset.ApplicationSetTemplateMeta{
					Name: "{{userGivenName}}",
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": store.Default.ManagedBy,
						"app.kubernetes.io/name":       "{{appName}}",
					},
				},
				Spec: appsetv1alpha1.ApplicationSpec{
					Source: appsetv1alpha1.ApplicationSource{
						RepoURL:        o.RepoURL,
						TargetRevision: o.Revision,
						Path:           o.FS.Join("kustomize", "{{appName}}", "overlays", o.Name),
					},
					Destination: appsetv1alpha1.ApplicationDestination{
						Server:    "{{destServer}}",
						Namespace: "{{destNamespace}}",
					},
					SyncPolicy: &appsetv1alpha1.SyncPolicy{
						Automated: &appsetv1alpha1.SyncPolicyAutomated{
							SelfHeal: true,
							Prune:    true,
						},
						SyncOptions: []string{
							"CreateNamespace=true",
						},
					},
				},
			},
		},
	}
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

func getBootstrapPaths(path, kustPath string) (string, error) {
	// for example: github.com/codefresh-io/argocd-autopilot/manifests
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return path, nil
	}

	// local file (in the filesystem)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	resourcePath, err := filepath.Rel(kustPath, absPath)
	if err != nil {
		return "", err
	}

	return resourcePath, nil
}
