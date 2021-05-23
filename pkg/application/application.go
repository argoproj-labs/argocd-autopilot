package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/argoproj/argocd-autopilot/pkg/fs"
	"github.com/argoproj/argocd-autopilot/pkg/git"
	"github.com/argoproj/argocd-autopilot/pkg/kube"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"

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
	ErrEmptyApp                     = errors.New("empty app not allowed")
	ErrEmptyAppName                 = errors.New("app name cannot be empty, please specify application name with: --app-name")
	ErrAppAlreadyInstalledOnProject = errors.New("application already installed on project")
	ErrAppCollisionWithExistingBase = errors.New("an application with the same name and a different base already exists, consider choosing a different name")
	ErrUnknownAppType               = errors.New("unknown application type")
)

type (
	Application interface {
		Name() string

		CreateFiles(repofs fs.FS, projectName string) error
	}

	Config struct {
		AppName           string `json:"appName"`
		UserGivenName     string `json:"userGivenName"`
		DestNamespace     string `json:"destNamespace"`
		DestServer        string `json:"destServer"`
		SrcPath           string `json:"srcPath"`
		SrcRepoURL        string `json:"srcRepoURL"`
		SrcTargetRevision string `json:"srcTargetRevision"`
	}

	CreateOptions struct {
		AppName          string
		AppType          string
		App              string
		DestNamespace    string
		DestServer       string
		InstallationMode string
	}

	baseApp struct {
		opts *CreateOptions
	}

	dirApp struct {
		baseApp
		config *Config
	}

	kustApp struct {
		baseApp
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
	cmd.Flags().StringVar(&co.App, "app", "", "The application specifier")
	cmd.Flags().StringVar(&co.AppType, "type", "", "The application type (kustomize|directory)")
	cmd.Flags().StringVar(&co.DestServer, "dest-server", store.Default.DestServer, fmt.Sprintf("K8s cluster URL (e.g. %s)", store.Default.DestServer))
	cmd.Flags().StringVar(&co.DestNamespace, "dest-namespace", "", "K8s target namespace (overrides the namespace specified in the kustomization.yaml)")
	cmd.Flags().StringVar(&co.InstallationMode, "installation-mode", InstallationModeNormal, "One of: normal|flat. "+
		"If flat, will commit the application manifests (after running kustomize build), otherwise will commit the kustomization.yaml")

	return co
}

// using heuristic from https://argoproj.github.io/argo-cd/user-guide/tool_detection/#tool-detection
func InferAppType(repofs fs.FS) string {
	if repofs.ExistsOrDie("app.yaml") && repofs.ExistsOrDie("components/params.libsonnet") {
		return "ksonnet"
	}

	if repofs.ExistsOrDie("Chart.yaml") {
		return "helm"
	}

	if repofs.ExistsOrDie("kustomization.yaml") || repofs.ExistsOrDie("kustomization.yml") || repofs.ExistsOrDie("Kustomization") {
		return "kustomize"
	}

	return "directory"
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
func (o *CreateOptions) Parse(ctx context.Context, co *git.CloneOptions, projectName string) (Application, error) {
	switch o.AppType {
	case "kustomize":
		return newKustApp(o, co, projectName)
	case "directory":
		return newDirApp(o), nil
	default:
		return nil, ErrUnknownAppType
	}
}

/* baseApp Application impl */
func (app *baseApp) Name() string {
	return app.opts.AppName
}

/* kustApp Application impl */
func newKustApp(o *CreateOptions, co *git.CloneOptions, projectName string) (*kustApp, error) {
	var err error
	app := &kustApp{
		baseApp: baseApp{o},
	}

	if o.App == "" {
		return nil, ErrEmptyApp
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
	if _, err := os.Stat(o.App); err == nil {
		log.G().Warn("using flat installation mode because base is a local file")
		o.InstallationMode = InstallationModeFlat
		o.App, err = filepath.Abs(o.App)
		if err != nil {
			return nil, err
		}
	}

	app.base = &kusttypes.Kustomization{
		TypeMeta: kusttypes.TypeMeta{
			APIVersion: kusttypes.KustomizationVersion,
			Kind:       kusttypes.KustomizationKind,
		},
		Resources: []string{o.App},
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

	if o.DestNamespace != "" && o.DestNamespace != "default" {
		app.overlay.Resources = append(app.overlay.Resources, "namespace.yaml")
		app.overlay.Namespace = o.DestNamespace
		app.namespace = kube.GenerateNamespace(o.DestNamespace)
	}

	app.config = &Config{
		AppName:           o.AppName,
		UserGivenName:     o.AppName,
		DestNamespace:     o.DestNamespace,
		DestServer:        o.DestServer,
		SrcRepoURL:        co.URL,
		SrcPath:           filepath.Join(store.Default.KustomizeDir, o.AppName, store.Default.OverlaysDir, projectName),
		SrcTargetRevision: co.Revision,
	}

	return app, nil
}

func (app *kustApp) CreateFiles(repofs fs.FS, projectName string) error {
	return kustCreateFiles(app, repofs, projectName)
}

func kustCreateFiles(app *kustApp, repofs fs.FS, projectName string) error {
	// create Base
	basePath := repofs.Join(store.Default.KustomizeDir, app.Name(), "base")
	baseKustomizationPath := repofs.Join(basePath, "kustomization.yaml")
	baseKustomizationYAML, err := yaml.Marshal(app.base)
	if err != nil {
		return fmt.Errorf("failed to marshal app base kustomization: %w", err)
	}

	if exists, err := writeFile(repofs, baseKustomizationPath, "base", baseKustomizationYAML); err != nil {
		return err
	} else if exists {
		// check if the bases are the same
		log.G().Debug("application base with the same name exists, checking for collisions")
		if collision, err := checkBaseCollision(repofs, baseKustomizationPath, app.base); err != nil {
			return err
		} else if collision {
			return ErrAppCollisionWithExistingBase
		}
	}

	// create Overlay
	overlayPath := repofs.Join(store.Default.KustomizeDir, app.Name(), "overlays", projectName)
	overlayKustomizationPath := repofs.Join(overlayPath, "kustomization.yaml")
	overlayKustomizationYAML, err := yaml.Marshal(app.overlay)
	if err != nil {
		return fmt.Errorf("failed to marshal app overlay kustomization: %w", err)
	}

	if exists, err := writeFile(repofs, overlayKustomizationPath, "overlay", overlayKustomizationYAML); err != nil {
		return err
	} else if exists {
		return ErrAppAlreadyInstalledOnProject
	}

	// get manifests - only used in flat installation mode
	if app.manifests != nil {
		manifestsPath := repofs.Join(basePath, "install.yaml")
		if _, err = writeFile(repofs, manifestsPath, "manifests", app.manifests); err != nil {
			return err
		}
	}

	// if we override the namespace we also need to write the namespace manifests next to the overlay
	if app.namespace != nil {
		nsPath := repofs.Join(overlayPath, "namespace.yaml")
		nsYAML, err := yaml.Marshal(app.namespace)
		if err != nil {
			return fmt.Errorf("failed to marshal app overlay namespace: %w", err)
		}

		if _, err = writeFile(repofs, nsPath, "application namespace", nsYAML); err != nil {
			return err
		}
	}

	configPath := repofs.Join(overlayPath, "config.json")
	config, err := json.Marshal(app.config)
	if err != nil {
		return fmt.Errorf("failed to marshal app config.json: %w", err)
	}

	if _, err = writeFile(repofs, configPath, "config", config); err != nil {
		return err
	}

	return nil
}

/* dirApp Application impl */
func newDirApp(opts *CreateOptions) *dirApp {
	app := &dirApp{
		baseApp: baseApp{opts},
	}

	host, orgRepo, path, gitRef, _, _, _ := util.ParseGitUrl(opts.App)

	app.config = &Config{
		AppName:           opts.AppName,
		UserGivenName:     opts.AppName,
		DestNamespace:     opts.DestNamespace,
		DestServer:        opts.DestServer,
		SrcRepoURL:        host + orgRepo,
		SrcPath:           path,
		SrcTargetRevision: gitRef,
	}

	return app
}

func (app *dirApp) CreateFiles(repofs fs.FS, projectName string) error {
	configPath := repofs.Join("apps", app.opts.AppName, projectName, "config.json")
	config, err := json.Marshal(app.config)
	if err != nil {
		return fmt.Errorf("failed to marshal app config.json: %w", err)
	}

	if _, err = writeFile(repofs, configPath, "config", config); err != nil {
		return err
	}

	return nil
}

func writeFile(repofs fs.FS, path, name string, data []byte) (bool, error) {
	absPath := repofs.Join(repofs.Root(), path)
	exists, err := repofs.CheckExistsOrWrite(path, data)
	if err != nil {
		return false, fmt.Errorf("failed to create '%s' file at '%s': %w", name, absPath, err)
	} else if exists {
		log.G().Infof("'%s' file exists in '%s'", name, absPath)
		return true, nil
	}

	log.G().Infof("created '%s' file at '%s'", name, absPath)
	return false, nil
}

func checkBaseCollision(repofs fs.FS, orgBasePath string, newBase *kusttypes.Kustomization) (bool, error) {
	orgBase := &kusttypes.Kustomization{}
	if err := repofs.ReadYamls(orgBasePath, orgBase); err != nil {
		return false, err
	}

	return !reflect.DeepEqual(orgBase, newBase), nil
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
