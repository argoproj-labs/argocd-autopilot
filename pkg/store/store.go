package store

import (
	"fmt"
	"runtime"
)

var s Store

var (
	binaryName = ""
	version    = "v99.99.99"
	gitCommit  = ""
)

type Version struct {
	Version   string
	GitCommit string
	GoVersion string
	Platform  string
}

type Store struct {
	BinaryName string
	Version    Version
	BaseGitURL string
}

// Get returns the global store
func Get() *Store {
	return &s
}

func init() {
	s.BinaryName = binaryName

	initVersion()
}

func initVersion() {
	s.Version.Version = version
	s.Version.GitCommit = gitCommit
	s.Version.GoVersion = runtime.Version()
	s.Version.Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
