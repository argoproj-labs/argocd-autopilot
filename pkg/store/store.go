package store

import (
	"fmt"
	"runtime"
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
