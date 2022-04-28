package application

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	"github.com/argoproj-labs/argocd-autopilot/pkg/kube"
	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	"github.com/ghodss/yaml"
	billyUtils "github.com/go-git/go-billy/v5/util"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/api/krusty"
	kusttypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

//go:generate mockgen -destination=./mocks/application.go -package=mocks -source=./application.go Application

const (
	InstallationModeFlat   = "flat"
	InstallationModeNormal = "normal"

	AppTypeKsonnet   = "ksonnet"
	AppTypeHelm      = "helm"
	AppTypeKustomize = "kustomize"
	AppTypeDirectory = "dir"
)

var (
	// Errors
	ErrEmptyAppSpecifier            = errors.New("empty app not allowed")
	ErrEmptyAppName                 = errors.New("app name can not be empty, please specify application name")
	ErrEmptyProjectName             = errors.New("project name can not be empty, please specificy project name with: --project")
	ErrAppAlreadyInstalledOnProject = errors.New("application already installed on project")
	ErrAppCollisionWithExistingBase = errors.New("an application with the same name and a different base already exists, consider choosing a different name")
	ErrUnknownAppType               = errors.New("unknown application type")
)

type (
	Application interface {
		Name() string

		CreateFiles(repofs fs.FS, appsfs fs.FS, projectName string) error
	}

	Config struct {
		AppName           string            `json:"appName"`
		UserGivenName     string            `json:"userGivenName"`
		DestNamespace     string            `json:"destNamespace"`
		DestServer        string            `json:"destServer"`
		SrcPath           string            `json:"srcPath"`
		SrcRepoURL        string            `json:"srcRepoURL"`
		SrcTargetRevision string            `json:"srcTargetRevision"`
		Labels            map[string]string `json:"labels"`
	}

	ClusterResConfig struct {
		Name   string `json:"name"`
		Server string `json:"server"`
	}

	CreateOptions struct {
		AppName          string
		AppType          string
		AppSpecifier     string
		DestNamespace    string
		DestServer       string
		InstallationMode string
		Labels           map[string]string
		Exclude          string
		Include          string
	}

	baseApp struct {
		opts *CreateOptions
	}

	dirApp struct {
		baseApp
		dirConfig *dirConfig
	}

	dirConfig struct {
		Config
		Exclude string `json:"exclude"`
		Include string `json:"include"`
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
	opts := &CreateOptions{}
	cmd.Flags().StringVar(&opts.AppSpecifier, "app", "", "The application specifier (e.g. github.com/argoproj/argo-workflows/manifests/cluster-install/?ref=v3.0.3)")
	cmd.Flags().StringVar(&opts.AppType, "type", "", "The application type (kustomize|dir)")
	cmd.Flags().StringVar(&opts.DestServer, "dest-server", store.Default.DestServer, fmt.Sprintf("K8s cluster URL (e.g. %s)", store.Default.DestServer))
	cmd.Flags().StringVar(&opts.DestNamespace, "dest-namespace", "", "K8s target namespace (overrides the namespace specified in the kustomization.yaml)")
	cmd.Flags().StringVar(&opts.InstallationMode, "installation-mode", InstallationModeNormal, "One of: normal|flat. "+
		"If flat, will commit the application manifests (after running kustomize build), otherwise will commit the kustomization.yaml")
	cmd.Flags().StringToStringVar(&opts.Labels, "labels", nil, "Optional labels that will be set on the Application resource. (e.g. \"{{ placeholder }}=my-org\"")
	cmd.Flags().StringVar(&opts.Include, "include", "", "Optional glob for files to include")
	cmd.Flags().StringVar(&opts.Exclude, "exclude", "", "Optional glob for files to exclude")

	return opts
}

// using heuristic from https://argoproj.github.io/argo-cd/user-guide/tool_detection/#tool-detection
func InferAppType(repofs fs.FS) string {
	if repofs.ExistsOrDie("app.yaml") && repofs.ExistsOrDie("components/params.libsonnet") {
		return AppTypeKsonnet
	}

	if repofs.ExistsOrDie("Chart.yaml") {
		return AppTypeHelm
	}

	if repofs.ExistsOrDie("kustomization.yaml") || repofs.ExistsOrDie("kustomization.yml") || repofs.ExistsOrDie("Kustomization") {
		return AppTypeKustomize
	}

	return AppTypeDirectory
}

func DeleteFromProject(repofs fs.FS, appName, projectName string) error {
	var dirToCheck string
	appDir := repofs.Join(store.Default.AppsDir, appName)
	appOverlay := repofs.Join(appDir, store.Default.OverlaysDir)
	if repofs.ExistsOrDie(appOverlay) {
		// kustApp
		dirToCheck = appOverlay
	} else {
		// dirApp
		dirToCheck = appDir
	}

	allProjects, err := repofs.ReadDir(dirToCheck)
	if err != nil {
		return fmt.Errorf("failed to check projects in '%s': %w", appName, err)
	}

	if !isInProject(allProjects, projectName) {
		return nil
	}

	var dirToRemove string
	if len(allProjects) == 1 {
		dirToRemove = appDir
		log.G().Infof("Deleting app '%s'", appName)
	} else {
		dirToRemove = repofs.Join(dirToCheck, projectName)
		log.G().Infof("Deleting app '%s' from project '%s'", appName, projectName)
	}

	err = billyUtils.RemoveAll(repofs, dirToRemove)
	if err != nil {
		return fmt.Errorf("failed to delete directory '%s': %w", dirToRemove, err)
	}

	return nil
}

func isInProject(allProjects []os.FileInfo, projectName string) bool {
	for _, project := range allProjects {
		if project.Name() == projectName {
			return true
		}
	}

	return false
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
func (o *CreateOptions) Parse(projectName, repoURL, targetRevision, repoRoot string) (Application, error) {
	switch o.AppType {
	case AppTypeKustomize:
		return newKustApp(o, projectName, repoURL, targetRevision, repoRoot)
	case AppTypeDirectory:
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
func newKustApp(o *CreateOptions, projectName, repoURL, targetRevision, repoRoot string) (*kustApp, error) {
	var err error
	app := &kustApp{
		baseApp: baseApp{o},
	}

	if o.AppSpecifier == "" {
		return nil, ErrEmptyAppSpecifier
	}

	if o.AppName == "" {
		return nil, ErrEmptyAppName
	}

	if projectName == "" {
		return nil, ErrEmptyProjectName
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

	if o.DestNamespace != "" && o.DestNamespace != "default" {
		app.overlay.Namespace = o.DestNamespace
		app.namespace = kube.GenerateNamespace(o.DestNamespace, nil)
	}

	app.config = &Config{
		AppName:           o.AppName,
		UserGivenName:     o.AppName,
		DestNamespace:     o.DestNamespace,
		DestServer:        o.DestServer,
		SrcRepoURL:        repoURL,
		SrcPath:           filepath.Join(repoRoot, store.Default.AppsDir, o.AppName, store.Default.OverlaysDir, projectName),
		SrcTargetRevision: targetRevision,
		Labels:            o.Labels,
	}

	return app, nil
}

func (app *kustApp) CreateFiles(repofs fs.FS, appsfs fs.FS, projectName string) error {
	return kustCreateFiles(app, repofs, appsfs, projectName)
}

func kustCreateFiles(app *kustApp, repofs fs.FS, appsfs fs.FS, projectName string) error {
	var err error

	// create Base
	appPath := appsfs.Join(store.Default.AppsDir, app.Name())
	basePath := appsfs.Join(appPath, "base")
	baseKustomizationPath := appsfs.Join(basePath, "kustomization.yaml")

	// check if app is in the same filesystem
	if appsfs.ExistsOrDie(appPath) {
		// check if the bases are the same
		log.G().Debug("application with the same name exists, checking for collisions")
		if collision, err := checkBaseCollision(appsfs, baseKustomizationPath, app.base); err != nil {
			return err
		} else if collision {
			return ErrAppCollisionWithExistingBase
		}
	} else if appsfs != repofs && repofs.ExistsOrDie(appPath) {
		appRepo, err := getAppRepo(repofs, app.Name())
		if err != nil {
			return fmt.Errorf("Failed getting app repo: %w", err)
		}

		return fmt.Errorf("an application with the same name already exists in '%s', consider choosing a different name", appRepo)
	}

	if err = appsfs.WriteYamls(baseKustomizationPath, app.base); err != nil {
		return err
	}

	// create Overlay
	overlayPath := appsfs.Join(appPath, "overlays", projectName)
	overlayKustomizationPath := appsfs.Join(overlayPath, "kustomization.yaml")
	if appsfs.ExistsOrDie(overlayKustomizationPath) {
		return ErrAppAlreadyInstalledOnProject
	}

	if err = appsfs.WriteYamls(overlayKustomizationPath, app.overlay); err != nil {
		return err
	}

	// create manifests - only used in flat installation mode
	if app.manifests != nil {
		manifestsPath := appsfs.Join(basePath, "install.yaml")
		if _, err = writeFile(appsfs, manifestsPath, "manifests", app.manifests); err != nil {
			return err
		}
	}

	clusterName, err := getClusterName(repofs, app.opts.DestServer)
	if err != nil {
		return err
	}

	if app.namespace != nil {
		if err = createNamespaceManifest(repofs, clusterName, app.namespace); err != nil {
			return err
		}
	}

	configPath := repofs.Join(overlayPath, "config.json")
	if repofs != appsfs {
		configPath = repofs.Join(appPath, projectName, "config.json")
	}

	if err = repofs.WriteJson(configPath, app.config); err != nil {
		return fmt.Errorf("failed to write app config.json: %w", err)
	}

	return nil
}

func getClusterName(repofs fs.FS, destServer string) (string, error) {
	// verify that the dest server already exists
	clusterName, err := serverToClusterName(repofs, destServer)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster name for the specified dest-server: %w", err)
	}
	if clusterName == "" {
		return "", fmt.Errorf("cluster '%s' is not configured yet, you need to create a project that uses this cluster first", destServer)
	}
	return clusterName, nil
}

func createNamespaceManifest(repofs fs.FS, clusterName string, namespace *v1.Namespace) error {
	nsYAML, err := yaml.Marshal(namespace)
	if err != nil {
		return fmt.Errorf("failed to marshal app overlay namespace: %w", err)
	}

	nsPath := repofs.Join(store.Default.BootsrtrapDir, store.Default.ClusterResourcesDir, clusterName, namespace.Name+"-ns.yaml")
	if _, err = writeFile(repofs, nsPath, "application namespace", nsYAML); err != nil {
		return err
	}
	return nil
}

/* dirApp Application impl */
func newDirApp(opts *CreateOptions) *dirApp {
	app := &dirApp{
		baseApp: baseApp{opts},
	}

	host, orgRepo, path, gitRef, _, suffix, _ := util.ParseGitUrl(opts.AppSpecifier)
	url := host + orgRepo + suffix
	if path == "" {
		path = "."
	}

	app.dirConfig = &dirConfig{
		Config: Config{
			AppName:           opts.AppName,
			UserGivenName:     opts.AppName,
			DestNamespace:     opts.DestNamespace,
			DestServer:        opts.DestServer,
			SrcRepoURL:        url,
			SrcPath:           path,
			SrcTargetRevision: gitRef,
			Labels:            opts.Labels,
		},
		Exclude: opts.Exclude,
		Include: opts.Include,
	}

	return app
}

func (app *dirApp) CreateFiles(repofs fs.FS, appsfs fs.FS, projectName string) error {
	appPath := repofs.Join(store.Default.AppsDir, app.opts.AppName, projectName)
	if repofs.ExistsOrDie(appPath) {
		return ErrAppAlreadyInstalledOnProject
	}

	configPath := repofs.Join(appPath, "config_dir.json")
	if err := repofs.WriteJson(configPath, app.dirConfig); err != nil {
		return fmt.Errorf("failed to write app config_dir.json: %w", err)
	}

	clusterName, err := getClusterName(repofs, app.opts.DestServer)
	if err != nil {
		return err
	}

	if app.opts.DestNamespace != "" && app.opts.DestNamespace != "default" {
		if err = createNamespaceManifest(repofs, clusterName, kube.GenerateNamespace(app.opts.DestNamespace, nil)); err != nil {
			return err
		}
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
// newKustDir.
func fixResourcesPaths(k *kusttypes.Kustomization, newKustDir string) error {
	for i, path := range k.Resources {
		// if path is a remote resource ignore it
		if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
			continue
		}

		absRes, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		k.Resources[i], err = filepath.Rel(newKustDir, absRes)
		log.G().WithFields(log.Fields{
			"from": absRes,
			"to":   k.Resources[i],
		}).Debug("adjusting kustomization paths to local filesystem")
		if err != nil {
			return err
		}
	}

	return nil
}

var generateManifests = func(k *kusttypes.Kustomization) ([]byte, error) {
	td, err := ioutil.TempDir(".", "auto-pilot")
	if err != nil {
		return nil, fmt.Errorf("failed creating temp dir: %w", err)
	}
	defer os.RemoveAll(td)

	absTd, err := filepath.Abs(td)
	if err != nil {
		return nil, fmt.Errorf("failed getting abs path for \"%s\": %w", td, err)
	}

	if err = fixResourcesPaths(k, absTd); err != nil {
		return nil, fmt.Errorf("failed fixing resources paths: %w", err)
	}

	kyaml, err := yaml.Marshal(k)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling yaml: %w", err)
	}

	kustomizationPath := filepath.Join(td, "kustomization.yaml")
	if err = ioutil.WriteFile(kustomizationPath, kyaml, 0400); err != nil {
		return nil, fmt.Errorf("failed writing file to \"%s\": %w", kustomizationPath, err)
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
		return nil, fmt.Errorf("failed running kustomization: %w", err)
	}

	return res.AsYaml()
}

var getAppRepo = func(repofs fs.FS, appName string) (string, error) {
	overlays, err := billyUtils.Glob(repofs, repofs.Join(store.Default.AppsDir, appName, store.Default.OverlaysDir, "**", "config.json"))
	if err != nil {
		return "", err
	}

	if len(overlays) == 0 {
		return "", fmt.Errorf("Application '%s' has no overlays", appName)
	}

	c := &Config{}
	return c.SrcRepoURL, repofs.ReadJson(overlays[0], c)
}
