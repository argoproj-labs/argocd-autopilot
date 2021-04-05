package application

import (
	"fmt"

	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/util"

	argocdutil "github.com/argoproj/argo-cd/v2/cmd/util"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
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
	tag     string
	name    string
	url     string
	path    string
	fs      filesys.FileSystem
	argoApp *v1alpha1.Application
}

func AddApplicationFlags(cmd *cobra.Command, defAppName string) *CreateOptions {
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

func (o *CreateOptions) Parse() (Application, error) {
	if o.AppSpecifier == "" {
		return nil, fmt.Errorf("empty app specifier not allowed")
	}

	app, err := argocdutil.ConstructApp("", o.AppName, []string{
		"app.kubernetes.io/managed-by=argo-autopilot",
		"app.kubernetes.io/name=" + o.AppName,
	}, []string{}, o.argoAppOptions, o.flags)
	if err != nil {
		return nil, err
	}

	// set default options
	app.Spec.SyncPolicy = &v1alpha1.SyncPolicy{
		Automated: &v1alpha1.SyncPolicyAutomated{
			SelfHeal: true,
			Prune:    true,
		},
	}

	return &application{
		path:    o.AppSpecifier, // TODO: supporting only path for now
		fs:      filesys.MakeFsOnDisk(),
		argoApp: app,
	}, nil
}

func (o *CreateOptions) ParseOrDie() Application {
	app, err := o.Parse()
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
