package application

import (
	"fmt"
	"strings"

	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/ghodss/yaml"

	argocdutil "github.com/argoproj/argo-cd/v2/cmd/util"
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
	url       string
	path      string
	fs        filesys.FileSystem
	argoApp   *v1alpha1.Application
}

type bootstrapApp struct {
	*application
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

		return &bootstrapApp{application: app}, nil
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
	secret := argocdsettings.Repository{
		URL: "github.com/whatever", //TODO: get real url
		UsernameSecret: &v1.SecretKeySelector{
			Key: "git_username",
		},
		PasswordSecret: &v1.SecretKeySelector{
			Key: "git_token",
		},
	}
	secret.UsernameSecret.Name = "autopilot-secrets"
	secret.PasswordSecret.Name = "autopilot-secrets"

	credentials, err := yaml.Marshal(secret)
	if err != nil {
		return nil, err
	}

	k := &kusttypes.Kustomization{
		TypeMeta: kusttypes.TypeMeta{
			APIVersion: kusttypes.KustomizationVersion,
			Kind:       kusttypes.KustomizationKind,
		},
		Resources: []string{app.path},
		ConfigMapGenerator: []kusttypes.ConfigMapArgs{
			{
				GeneratorArgs: kusttypes.GeneratorArgs{
					Name:     "argocd-cm",
					Behavior: kusttypes.BehaviorMerge.String(),
					KvPairSources: kusttypes.KvPairSources{
						LiteralSources: []string{
							"repository.credentials=" + string(credentials),
						},
					},
				},
			},
		},
		Namespace: app.namespace,
	}

	res, err := runKustomization(k)
	if err != nil {
		return nil, err
	}

	return util.JoinManifests(res, createNamespace(app.namespace)), nil
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

func runKustomization(k *kusttypes.Kustomization) ([]byte, error) {
	k.FixKustomizationPostUnmarshalling()
	errs := k.EnforceFields()
	if len(errs) > 0 {
		return nil, fmt.Errorf("kustomization errors: %s", strings.Join(errs, "\n"))
	}
	k.FixKustomizationPreMarshalling()

	kyaml, err := yaml.Marshal(k)
	if err != nil {
		return nil, err
	}

	kust := krusty.MakeKustomizer(&krusty.Options{})
	fs := filesys.MakeFsInMemory()
	f, err := fs.Create("kustomization.yaml")
	if err != nil {
		return nil, err
	}

	_, err = f.Write(kyaml)
	if err != nil {
		return nil, err
	}

	res, err := kust.Run(fs, "kustomization.yaml")
	if err != nil {
		return nil, err
	}

	return res.AsYaml()
}
