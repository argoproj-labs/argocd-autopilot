package application

import (
	"fmt"

	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/spf13/pflag"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
)

type Application interface {
	GenerateManifests() ([]byte, error)
}

type CreateOptions struct {
	AppSpecifier string
}

type application struct {
	tag  string
	name string
	url  string
	path string
	fs   filesys.FileSystem
}

func AddApplicationFlags(flags *pflag.FlagSet) *CreateOptions {
	co := &CreateOptions{}
	flags.StringVar(&co.AppSpecifier, "app", "", "The application specifier")

	return co
}

func (app *application) GenerateManifests() ([]byte, error) {
	return app.kustomizeBuild() // TODO: supporting only kustomize for now
}

func (o *CreateOptions) Parse() (Application, error) {
	if o.AppSpecifier == "" {
		return nil, fmt.Errorf("empty app specifier not allowed")
	}
	return &application{
		path: o.AppSpecifier, // TODO: supporting only path for now
		fs:   filesys.MakeFsOnDisk(),
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
