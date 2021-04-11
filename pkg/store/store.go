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

var Common = struct {
	ArgoCDName    string
	BootsrtrapDir string
	DummyName     string
	EnvsDir       string
	KustomizeDir  string
	ManagedBy     string
	RootName      string
	SecretName    string
	Username      string
	WaitInterval  time.Duration
}{
	ArgoCDName:    "argo-cd",
	BootsrtrapDir: "bootstrap",
	DummyName:     "DUMMY",
	EnvsDir:       "envs",
	KustomizeDir:  "kustomize",
	ManagedBy:     "argo-autopilot",
	RootName:      "root",
	SecretName:    "autopilot-secret",
	Username:      "username",
	WaitInterval:  time.Second * 3,
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
