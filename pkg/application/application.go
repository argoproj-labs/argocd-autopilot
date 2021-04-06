package application

import (
	"fmt"

	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/ghodss/yaml"

	argocdutil "github.com/argoproj/argo-cd/v2/cmd/util"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdsettings "github.com/argoproj/argo-cd/v2/util/settings"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
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
	application
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

	if bootstrap {
		return &bootstrapApp{
			path:      o.AppSpecifier, // TODO: supporting only path for now
			namespace: namespace,
			fs:        filesys.MakeFsOnDisk(),
			argoApp:   argoApp,
		}, nil
	}

	return &application{
		path:      o.AppSpecifier, // TODO: supporting only path for now
		namespace: namespace,
		fs:        filesys.MakeFsOnDisk(),
		argoApp:   argoApp,
	}, nil
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
		UsernameSecret: &corev1.SecretKeySelector{
			Key: "git_username",
		},
		PasswordSecret: &corev1.SecretKeySelector{
			Key: "git_token",
		},
	}
	secret.UsernameSecret.Name = "autopilot-secrets"
	secret.PasswordSecret.Name = "autopilot-secrets"
	credentials, err := yaml.Marshal(secret)
	if err != nil {
		return nil, err
	}

	cm := kusttypes.ConfigMapArgs{}
	cm.Name = "argocd-cm"
	cm.Behavior = kusttypes.BehaviorMerge.String()
	cm.LiteralSources = []string{
		"repository.credentials=" + string(credentials),
	}

	k := &kusttypes.Kustomization{
		TypeMeta: kusttypes.TypeMeta{
			APIVersion: kusttypes.KustomizationVersion,
			Kind:       kusttypes.KustomizationKind,
		},
		Resources: []string{app.path},
		ConfigMapGenerator: []kusttypes.ConfigMapArgs{
			cm,
		},
		Namespace: "argocd", // TODO: replace with namespace value
	}

	return nil, nil

}

func getLabels(appName string) []string {
	return []string{
		"app.kubernetes.io/managed-by=argo-autopilot",
		"app.kubernetes.io/name=" + appName,
	}
}
