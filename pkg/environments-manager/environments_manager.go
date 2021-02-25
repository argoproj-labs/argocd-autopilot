package environments_manager

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/codefresh-io/cf-argo/pkg/helpers"
	"github.com/codefresh-io/cf-argo/pkg/kube"
	"github.com/codefresh-io/cf-argo/pkg/store"
	"github.com/ghodss/yaml"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kustomize "sigs.k8s.io/kustomize/api/types"
)

// errors
var (
	ErrEnvironmentAlreadyExists = errors.New("environment already exists")
	ErrEnvironmentNotExist      = errors.New("environment does not exist")
	ErrAppNotFound              = errors.New("app not found")

	ConfigFileName = fmt.Sprintf("%s.yaml", store.AppName)

	yamlSeparator = regexp.MustCompile(`\n---`)
)

const (
	configVersion   = "1.0"
	labelsManagedBy = "app.kubernetes.io/managed-by"
	labelsName      = "app.kubernetes.io/name"
)

type (
	Config struct {
		path         string                  // the path from which the config was loaded
		Version      string                  `json:"version"`
		Environments map[string]*Environment `json:"environments"`
	}

	Environment struct {
		c                   *Config
		name                string
		RootApplicationPath string `json:"rootAppPath"`
		TemplateRef         string `json:"templateRef"`
	}

	Application struct {
		*v1alpha1.Application
		Path string
	}
)

func NewConfig(path string) *Config {
	return &Config{
		path:         path,
		Version:      configVersion,
		Environments: make(map[string]*Environment),
	}
}

// Persist saves the config to file
func (c *Config) Persist() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(c.path, ConfigFileName), data, 0644)
}

// AddEnvironmentP adds a new environment, copies all of the argocd apps to the relative
// location in the repository that c is managing, and persists the config object
func (c *Config) AddEnvironmentP(ctx context.Context, env *Environment, values interface{}, dryRun bool) error {
	if _, exists := c.Environments[env.name]; exists {
		return fmt.Errorf("%w: %s", ErrEnvironmentAlreadyExists, env.name)
	}

	// copy all of the argocd apps to the correct location in the destination repo
	newEnv, err := c.installEnv(env)
	if err != nil {
		return err
	}

	c.Environments[env.name] = newEnv
	if err = c.Persist(); err != nil {
		return err
	}

	cs, err := store.Get().NewKubeClient(ctx).KubernetesClientSet()
	if err != nil {
		return err
	}

	_, err = cs.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-argocd", env.name)},
	}, metav1.CreateOptions{})
	if err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return err
		}
	}

	manifests, err := kube.KustBuild(newEnv.bootstrapUrl(), values)
	if err != nil {
		return err
	}

	return store.Get().NewKubeClient(ctx).Apply(ctx, &kube.ApplyOptions{
		Manifests: manifests,
		DryRun:    dryRun,
	})
}

// DeleteEnvironmentP deletes an environment and persists the config object
func (c *Config) DeleteEnvironmentP(ctx context.Context, name string, values interface{}, dryRun bool) error {
	env, exists := c.Environments[name]
	if !exists {
		return ErrEnvironmentNotExist
	}

	err := env.cleanup()
	if err != nil {
		return err
	}

	delete(c.Environments, name)
	if err = c.Persist(); err != nil {
		return err
	}

	manifests, err := kube.KustBuild(env.bootstrapUrl(), values)
	if err != nil {
		return err
	}

	return store.Get().NewKubeClient(ctx).Delete(ctx, &kube.DeleteOptions{
		Manifests: manifests,
		DryRun:    dryRun,
	})
}

func (c *Config) FirstEnv() *Environment {
	for _, env := range c.Environments {
		return env
	}
	return nil
}

// LoadConfig loads the config from the specified path
func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(filepath.Join(path, ConfigFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file does not exist: %s", path)
		}
		return nil, err
	}

	c := new(Config)
	c.path = path
	if err = yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}
	for name, e := range c.Environments {
		e.c = c
		e.name = name
	}

	return c, nil
}

func (c *Config) installEnv(env *Environment) (*Environment, error) {
	lapps, err := env.leafApps()
	if err != nil {
		return nil, err
	}

	newEnv := &Environment{
		name:                env.name,
		c:                   c,
		TemplateRef:         env.TemplateRef,
		RootApplicationPath: env.RootApplicationPath,
	}
	for _, la := range lapps {
		if err = newEnv.installApp(env.c.path, la); err != nil {
			return nil, err
		}
	}

	// copy the tpl "argocd-apps" to the matching dir in the dst repo
	src := filepath.Join(env.c.path, filepath.Dir(env.RootApplicationPath))
	var dstApplicationPath string
	if len(c.Environments) == 0 {
		dstApplicationPath = newEnv.RootApplicationPath
	} else {
		dstApplicationPath = c.FirstEnv().RootApplicationPath
	}

	dst := filepath.Join(c.path, filepath.Dir(dstApplicationPath))
	err = helpers.CopyDir(src, dst)
	if err != nil {
		return nil, err
	}

	return newEnv, nil
}

func (c *Config) getApp(appName string) (*Application, error) {
	err := ErrAppNotFound
	var app *Application

	for _, e := range c.Environments {
		app, err = e.GetApp(appName)
		if err != nil && !errors.Is(err, ErrAppNotFound) {
			return nil, err
		}

		if app != nil {
			return app, nil
		}
	}

	return app, err
}

func (e *Environment) UpdateTemplateRef(templateRef string) {
	e.TemplateRef = templateRef
}

func (e *Environment) bootstrapUrl() string {
	var parts []string

	switch {
	case strings.Contains(e.TemplateRef, "#"):
		parts = strings.Split(e.TemplateRef, "#")
	case strings.Contains(e.TemplateRef, "@"):
		parts = strings.Split(e.TemplateRef, "@")
	default:
		parts = []string{e.TemplateRef}
	}

	bootstrapUrl := fmt.Sprintf("%s/bootstrap", parts[0])
	if len(parts) > 1 {
		return fmt.Sprintf("%s?ref=%s", bootstrapUrl, parts[1])
	}

	return bootstrapUrl
}

func (e *Environment) cleanup() error {
	rootApp, err := e.GetRootApp()
	if err != nil {
		return err
	}

	return rootApp.deleteFromFilesystem(e.c.path)
}

func (e *Environment) installApp(srcRootPath string, app *Application) error {
	appName := app.labelName()
	refApp, err := e.c.getApp(appName)
	if err != nil {
		if !errors.Is(err, ErrAppNotFound) {
			return err
		}

		return e.installNewApp(srcRootPath, app)
	}

	baseLocation, err := refApp.getBaseLocation(e.c.path)
	if err != nil {
		return err
	}

	absSrc := filepath.Join(srcRootPath, app.srcPath())

	dst := filepath.Clean(filepath.Join(baseLocation, "..", "overlays", e.name))
	absDst := filepath.Join(e.c.path, dst)

	err = helpers.CopyDir(absSrc, absDst)
	if err != nil {
		return err
	}

	app.setSrcPath(dst)
	return app.save()
}

func (e *Environment) installNewApp(srcRootPath string, app *Application) error {
	appFolder := filepath.Clean(filepath.Join(app.srcPath(), "..", ".."))
	absSrc := filepath.Join(srcRootPath, appFolder)
	absDst := filepath.Join(e.c.path, appFolder)

	return helpers.CopyDir(absSrc, absDst)
}

// Uninstall removes all managed apps and returns true if there are no more
// apps left in the environment.
func (e *Environment) Uninstall() (bool, error) {
	rootApp, err := e.GetRootApp()
	if err != nil {
		return false, err
	}

	uninstalled, err := rootApp.uninstall(e.c.path)
	if uninstalled {
		return true, createDummy(filepath.Join(e.c.path, rootApp.srcPath()))
	}

	return false, err
}

func (e *Environment) leafApps() ([]*Application, error) {
	rootApp, err := e.GetRootApp()
	if err != nil {
		return nil, err
	}

	return rootApp.leafApps(e.c.path)
}

func (e *Environment) GetRootApp() (*Application, error) {
	return getAppFromFile(filepath.Join(e.c.path, e.RootApplicationPath))
}

func (e *Environment) GetApp(appName string) (*Application, error) {
	rootApp, err := e.GetRootApp()
	if err != nil {
		return nil, err
	}

	app, err := e.getAppRecurse(rootApp, appName)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, fmt.Errorf("%w: %s", ErrAppNotFound, appName)
	}
	return app, nil
}

func (e *Environment) getAppRecurse(root *Application, appName string) (*Application, error) {
	if root.labelName() == appName {
		return root, nil
	}

	appsDir := root.srcPath() // check if it's not in this repo
	filenames, err := filepath.Glob(filepath.Join(e.c.path, appsDir, "*.yaml"))
	if err != nil {
		return nil, err
	}

	for _, f := range filenames {
		app, err := getAppFromFile(f)
		if err != nil || app == nil {
			// not an argocd app - ignore
			continue
		}

		if !app.isManaged() {
			continue
		}

		res, err := e.getAppRecurse(app, appName)
		if err != nil || res != nil {
			return res, err
		}
	}

	return nil, nil
}

func getAppFromFile(path string) (*Application, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	for _, text := range yamlSeparator.Split(string(data), -1) {
		if strings.TrimSpace(text) == "" {
			continue
		}
		u := &unstructured.Unstructured{}
		err := yaml.Unmarshal([]byte(text), u)
		if err != nil {
			return nil, err
		}

		if u.GetKind() == "Application" {
			app := &v1alpha1.Application{}
			if err := yaml.Unmarshal(data, app); err != nil {
				return nil, err
			}

			return &Application{app, path}, nil
		}
	}

	return nil, nil
}

func (a *Application) deleteFromFilesystem(rootPath string) error {
	srcDir := filepath.Join(rootPath, a.srcPath())
	err := os.RemoveAll(srcDir)
	if err != nil {
		return err
	}

	projectPath := filepath.Join(filepath.Dir(a.Path), fmt.Sprintf("%s-project.yaml", a.Name))
	err = os.Remove(projectPath)
	if err != nil {
		return err
	}

	err = os.Remove(a.Path)
	if err != nil {
		return err
	}

	return nil
}

func (a *Application) srcPath() string {
	return a.Spec.Source.Path
}

func (a *Application) setSrcPath(newPath string) {
	a.Spec.Source.Path = newPath
}

func (a *Application) labelName() string {
	return a.labelValue(labelsName)
}

func (a *Application) isManaged() bool {
	return a.labelValue(labelsManagedBy) == store.AppName
}

func (a *Application) labelValue(label string) string {
	if a.Labels == nil {
		return ""
	}

	return a.Labels[label]
}

func (a *Application) getBaseLocation(absRoot string) (string, error) {
	refKust := filepath.Join(absRoot, a.srcPath(), "kustomization.yaml")
	bytes, err := ioutil.ReadFile(refKust)
	if err != nil {
		return "", err
	}

	k := &kustomize.Kustomization{}
	err = yaml.Unmarshal(bytes, k)
	if err != nil {
		return "", err
	}

	return filepath.Clean(filepath.Join(a.srcPath(), k.Resources[0])), nil
}

func (a *Application) save() error {
	data, err := yaml.Marshal(a)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(a.Path, data, 0644)
}

func (a *Application) leafApps(rootPath string) ([]*Application, error) {
	childApps, err := a.childApps(rootPath)
	if err != nil {
		return nil, err
	}

	isLeaf := true
	res := []*Application{}
	for _, childApp := range childApps {
		isLeaf = false
		if childApp.isManaged() {
			childRes, err := childApp.leafApps(rootPath)
			if err != nil {
				return nil, err
			}

			res = append(res, childRes...)
		}
	}

	if isLeaf && a.isManaged() {
		res = append(res, a)
	}

	return res, nil
}

func (a *Application) uninstall(rootPath string) (bool, error) {
	uninstalled := false
	childApps, err := a.childApps(rootPath)
	if err != nil {
		return uninstalled, err
	}

	totalUninstalled := 0
	for _, childApp := range childApps {
		if childApp.isManaged() {
			childUninstalled, err := childApp.uninstall(rootPath)
			if err != nil {
				return uninstalled, err
			}

			if childUninstalled {
				err = os.Remove(childApp.Path)
				if err != nil {
					return uninstalled, err
				}

				totalUninstalled++
			}
		}
	}

	uninstalled = len(childApps) == totalUninstalled
	return uninstalled, nil
}

func (a *Application) childApps(rootPath string) ([]*Application, error) {
	filenames, err := filepath.Glob(filepath.Join(rootPath, a.srcPath(), "*.yaml"))
	if err != nil {
		return nil, err
	}

	res := []*Application{}
	for _, f := range filenames {
		childApp, err := getAppFromFile(f)
		if err != nil {
			fmt.Printf("file is not an argo-cd application manifest %s\n", f)
			continue
		}

		if childApp != nil {
			res = append(res, childApp)
		}
	}

	return res, nil
}

func createDummy(path string) error {
	file, err := os.Create(filepath.Join(path, "DUMMY"))
	if err != nil {
		return err
	}
	return file.Close()
}
