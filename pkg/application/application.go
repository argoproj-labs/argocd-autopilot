package application

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/argoproj/argocd-autopilot/pkg/kube"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
	kusttypes "sigs.k8s.io/kustomize/api/types"
)

//go:generate mockery -name Application -filename application.go

const (
	InstallationModeFlat   = "flat"
	InstallationModeNormal = "normal"
)

var (
	// Errors
	ErrEmptyAppSpecifier = errors.New("empty app specifier not allowed")
	ErrEmptyAppName      = errors.New("app name cannot be empty, please specify application name with: --app-name")
)

type (
	Application interface {
		Name() string

		// Base returns the base kustomization file for this app.
		Base() *kusttypes.Kustomization

		// Manifests returns all of the applications manifests in case flat installation mode is used
		Manifests() []byte

		// Overlay returns the overlay kustomization object that is looking on this
		// app.Base() file.
		Overlay() *kusttypes.Kustomization

		// Namespace returns a Namespace object for the application's namespace
		Namespace() *v1.Namespace

		// Config returns this app's config.json file that should be next to the overlay
		// kustomization.yaml file. This is used by the environment's application set
		// to generate the final argo-cd application.
		Config() *Config
	}

	Config struct {
		AppName       string `json:"appName,omitempty"`
		UserGivenName string `json:"userGivenName,omitempty"`
		DestNamespace string `json:"destNamespace,omitempty"`
		DestServer    string `json:"destServer,omitempty"`
	}

	CreateOptions struct {
		AppSpecifier     string
		AppName          string
		DestNamespace    string
		DestServer       string
		InstallationMode string
	}

	application struct {
		opts      *CreateOptions
		base      *kusttypes.Kustomization
		overlay   *kusttypes.Kustomization
		manifests []byte
		namespace *v1.Namespace
		config    *Config
	}
)

// AddFlags adds application creation flags to cmd.
func AddFlags(cmd *cobra.Command) *CreateOptions {
	co := &CreateOptions{}

	cmd.Flags().StringVar(&co.AppSpecifier, "app", "", "The application specifier (e.g. argocd@v1.0.2)")
	cmd.Flags().StringVar(&co.DestServer, "dest-server", store.Default.DestServer, fmt.Sprintf("K8s cluster URL (e.g. %s)", store.Default.DestServer))
	cmd.Flags().StringVar(&co.DestNamespace, "dest-namespace", "", "K8s target namespace (overrides the namespace specified in the kustomization.yaml)")
	cmd.Flags().StringVar(&co.InstallationMode, "installation-mode", InstallationModeNormal, "One of: normal|flat. "+
		"If flat, will commit the application manifests (after running kustomize build), otherwise will commit the kustomization.yaml")

	return co
}

// GenerateManifests writes the in-memory kustomization to disk, fixes relative resources and
// runs kustomize build, then returns the generated manifests.
//
// If there is a namespace on 'k' a namespace.yaml file with the namespace object will be
// written next to the persisted kustomization.yaml.
//
// To include the namespace in the generated
// manifests just add 'namespace.yaml' to the resources of the kustomization
func GenerateManifests(k *kusttypes.Kustomization) ([]byte, error) {
	return generateManifests(k)
}

/* CreateOptions impl */
// Parse tries to parse `CreateOptions` into an `Application`.
func (o *CreateOptions) Parse() (Application, error) {
	return parseApplication(o)
}

/* Application impl */
func (app *application) Name() string {
	return app.opts.AppName
}

func (app *application) Base() *kusttypes.Kustomization {
	return app.base
}

func (app *application) Overlay() *kusttypes.Kustomization {
	return app.overlay
}

func (app *application) Namespace() *v1.Namespace {
	return app.namespace
}

func (app *application) Config() *Config {
	return app.config
}

func (app *application) Manifests() []byte {
	return app.manifests
}

// fixResourcesPaths adjusts all relative paths in the kustomization file to the specified
// `kustomizationPath`.
func fixResourcesPaths(k *kusttypes.Kustomization, kustomizationPath string) error {
	for i, path := range k.Resources {
		// if path is a local file
		if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
			continue
		}

		ap, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		k.Resources[i], err = filepath.Rel(kustomizationPath, ap)
		if err != nil {
			return err
		}
	}

	return nil
}

func parseApplication(o *CreateOptions) (*application, error) {
	var err error
	app := &application{opts: o}

	if o.AppSpecifier == "" {
		return nil, ErrEmptyAppSpecifier
	}

	if o.AppName == "" {
		return nil, ErrEmptyAppName
	}

	switch o.InstallationMode {
	case InstallationModeFlat, InstallationModeNormal:
	case "":
		o.InstallationMode = InstallationModeNormal
	default:
		return nil, fmt.Errorf("unknown installation mode: %s", o.InstallationMode)
	}

	// if app specifier is a local file
	if _, err := os.Stat(o.AppSpecifier); err == nil {
		log.G().Warn("using flat installation mode because base is a local file")
		o.InstallationMode = InstallationModeFlat
		o.AppSpecifier, err = filepath.Abs(o.AppSpecifier)
		if err != nil {
			return nil, err
		}
	}

	app.base = &kusttypes.Kustomization{
		TypeMeta: kusttypes.TypeMeta{
			APIVersion: kusttypes.KustomizationVersion,
			Kind:       kusttypes.KustomizationKind,
		},
		Resources: []string{o.AppSpecifier},
	}

	if o.InstallationMode == InstallationModeFlat {
		log.G().Info("building manifests...")
		app.manifests, err = generateManifests(app.base)
		if err != nil {
			return nil, err
		}

		app.base.Resources[0] = "install.yaml"
	}

	app.overlay = &kusttypes.Kustomization{
		Resources: []string{
			"../../base",
		},
		TypeMeta: kusttypes.TypeMeta{
			APIVersion: kusttypes.KustomizationVersion,
			Kind:       kusttypes.KustomizationKind,
		},
	}

	if o.DestNamespace != "" {
		app.overlay.Resources = append(app.overlay.Resources, "namespace.yaml")
		app.overlay.Namespace = o.DestNamespace
		app.namespace = kube.GenerateNamespace(o.DestNamespace)
	}

	app.config = &Config{
		AppName:       o.AppName,
		UserGivenName: o.AppName,
		DestNamespace: o.DestNamespace,
		DestServer:    o.DestServer,
	}

	return app, nil
}

var generateManifests = func(k *kusttypes.Kustomization) ([]byte, error) {
	td, err := ioutil.TempDir("", "auto-pilot")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(td)

	kustomizationPath := filepath.Join(td, "kustomization.yaml")
	if err = fixResourcesPaths(k, kustomizationPath); err != nil {
		return nil, err
	}

	kyaml, err := yaml.Marshal(k)
	if err != nil {
		return nil, err
	}

	if err = ioutil.WriteFile(kustomizationPath, kyaml, 0400); err != nil {
		return nil, err
	}

	if k.Namespace != "" {
		log.G().Debug("detected namespace on kustomization, generating namespace.yaml file")
		ns, err := yaml.Marshal(kube.GenerateNamespace(k.Namespace))
		if err != nil {
			return nil, err
		}
		if err = ioutil.WriteFile(filepath.Join(td, "namespace.yaml"), ns, 0400); err != nil {
			return nil, err
		}
	}

	log.G().WithFields(log.Fields{
		"bootstrapKustPath": kustomizationPath,
		"resourcePath":      k.Resources[0],
	}).Debugf("running bootstrap kustomization: %s\n", string(kyaml))

	opts := krusty.MakeDefaultOptions()
	opts.DoLegacyResourceSort = true
	kust := krusty.MakeKustomizer(opts)
	fs := filesys.MakeFsOnDisk()
	res, err := kust.Run(fs, td)
	if err != nil {
		return nil, err
	}

	return res.AsYaml()
}
