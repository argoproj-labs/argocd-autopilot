package environments_manager

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/codefresh-io/cf-argo/pkg/helpers"
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kustomize "sigs.k8s.io/kustomize/api/types"
)

// errors
var (
	ErrEnvironmentAlreadyExists = errors.New("environment already exists")
	ErrEnvironmentNotExist      = errors.New("environment does not exist")
	ErrAppNotFound              = errors.New("app not found")

	yamlSeparator = regexp.MustCompile(`\n---`)
)

const (
	configVersion = "1.0"

	configFileName  = "codefresh.yaml"
	labelsCfName    = "cf-name"
	labelsManagedBy = "ent-managed-by"
	DefaultAppsPath = "argocd-apps"
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
	}

	Application struct {
		*v1alpha1.Application
		path string
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

	return ioutil.WriteFile(filepath.Join(c.path, configFileName), data, 0644)
}

// AddEnvironmentP adds a new environment, copies all of the argocd apps to the relative
// location in the repository that c is managing, and persists the config object
func (c *Config) AddEnvironmentP(env *Environment) error {
	if _, exists := c.Environments[env.name]; exists {
		return fmt.Errorf("%w: %s", ErrEnvironmentAlreadyExists, env.name)
	}

	// copy all of the argocd apps to the correct location in the destination repo
	newEnv, err := c.installEnv(env)
	if err != nil {
		return err
	}

	c.Environments[env.name] = newEnv
	return c.Persist()
}

// DeleteEnvironmentP deletes an environment and persists the config object
func (c *Config) DeleteEnvironmentP(name string, env Environment) error {
	if _, exists := c.Environments[name]; !exists {
		return ErrEnvironmentNotExist
	}

	delete(c.Environments, name)

	return c.Persist()
}

func (c *Config) FirstEnv() *Environment {
	for _, env := range c.Environments {
		return env
	}
	return nil
}

// LoadConfig loads the config from the specified path
func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(filepath.Join(path, configFileName))
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
		RootApplicationPath: env.RootApplicationPath,
	}
	for _, la := range lapps {
		if err = newEnv.installApp(env.c.path, la); err != nil {
			return nil, err
		}
	}

	// copy the tpl "argocd-apps" to the matching dir in the dst repo
	src := filepath.Join(env.c.path, filepath.Dir(env.RootApplicationPath))
	dst := filepath.Join(c.path, filepath.Dir(c.FirstEnv().RootApplicationPath))
	err = helpers.CopyDir(src, dst)
	if err != nil {
		return nil, err
	}

	return newEnv, nil
}

func (c *Config) getAppByName(appName string) (*Application, error) {
	var err error
	var app *Application

	for _, e := range c.Environments {
		app, err = e.getAppByName(appName)
		if err != nil && !errors.Is(err, ErrAppNotFound) {
			return nil, err
		}
		if app != nil {
			return app, nil
		}
	}

	return app, err
}

func (e *Environment) installApp(srcRootPath string, app *Application) error {
	appName := app.cfName()
	refApp, err := e.c.getAppByName(appName)
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

func (e *Environment) leafApps() ([]*Application, error) {
	rootApp, err := e.getRootApp()
	if err != nil {
		return nil, err
	}

	return e.leafAppsRecurse(rootApp)
}

func (e *Environment) leafAppsRecurse(root *Application) ([]*Application, error) {
	filenames, err := filepath.Glob(filepath.Join(e.c.path, root.Spec.Source.Path, "*.yaml"))
	if err != nil {
		return nil, err
	}

	isLeaf := true
	res := []*Application{}
	for _, f := range filenames {
		childApp, err := getAppFromFile(f)
		if err != nil {
			fmt.Printf("file is not an argo-cd application manifest %s\n", f)
			continue
		}

		if childApp != nil {
			isLeaf = false
			childRes, err := e.leafAppsRecurse(childApp)
			if err != nil {
				return nil, err
			}
			res = append(res, childRes...)
		}
	}
	if isLeaf {
		res = append(res, root)
	}

	return res, nil
}

func (e *Environment) getRootApp() (*Application, error) {
	return getAppFromFile(filepath.Join(e.c.path, e.RootApplicationPath))
}

func (e *Environment) getAppByName(appName string) (*Application, error) {
	rootApp, err := e.getRootApp()
	if err != nil {
		return nil, err
	}

	app, err := e.getAppByNameRecurse(rootApp, appName)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, fmt.Errorf("%w: %s", ErrAppNotFound, appName)
	}
	return app, nil
}

func (e *Environment) getAppByNameRecurse(root *Application, appName string) (*Application, error) {
	if root.cfName() == appName {
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

		if !app.isManagedBy() {
			continue
		}

		res, err := e.getAppByNameRecurse(app, appName)
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

func (a *Application) srcPath() string {
	return a.Spec.Source.Path
}

func (a *Application) setSrcPath(newPath string) {
	a.Spec.Source.Path = newPath
}

func (a *Application) cfName() string {
	return a.labelValue(labelsCfName)
}

func (a *Application) isManagedBy() bool {
	return a.labelValue(labelsManagedBy) == "codefresh.io"
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

	return ioutil.WriteFile(a.path, data, 0644)
}
