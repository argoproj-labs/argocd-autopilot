package store

import (
	"fmt"
	"runtime"
	"time"
)

var s Store

var (
	binaryName                         = ""
	version                            = "v99.99.99"
	buildDate                          = ""
	gitCommit                          = ""
	installationManifestsURL           = "manifests"
	installationManifestsNamespacedURL = "manifests/namespace-install"
)

type Version struct {
	Version    string
	BuildDate  string
	GitCommit  string
	GoVersion  string
	GoCompiler string
	Platform   string
}

type Store struct {
	BinaryName                         string
	Version                            Version
	InstallationManifestsURL           string
	InstallationManifestsNamespacedURL string
}

var Default = struct {
	BootsrtrapDir       string
	KustomizeDir        string
	OverlaysDir         string
	BaseDir             string
	ArgoCDName          string
	BootsrtrapAppName   string
	DummyName           string
	EnvsDir             string
	ManagedBy           string
	RootAppName         string
	RepoCredsSecretName string
	GitUsername         string
	WaitInterval        time.Duration
}{
	KustomizeDir:        "kustomize",
	BootsrtrapDir:       "bootstrap",
	OverlaysDir:         "overlays",
	BaseDir:             "base",
	ArgoCDName:          "argo-cd",
	BootsrtrapAppName:   "autopilot-bootstrap",
	DummyName:           "DUMMY",
	EnvsDir:             "envs",
	ManagedBy:           "argo-autopilot",
	RootAppName:         "root",
	RepoCredsSecretName: "autopilot-secret",
	GitUsername:         "username",
	WaitInterval:        time.Second * 3,
}

// Get returns the global store
func Get() *Store {

	return &s
}

func init() {
	s.BinaryName = binaryName
	s.InstallationManifestsURL = installationManifestsURL
	s.InstallationManifestsNamespacedURL = installationManifestsNamespacedURL

	initVersion()
}

func initVersion() {
	s.Version.Version = version
	s.Version.BuildDate = buildDate
	s.Version.GitCommit = gitCommit
	s.Version.GoVersion = runtime.Version()
	s.Version.GoCompiler = runtime.Compiler
	s.Version.Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
