package store

import (
	"context"
	"fmt"
	"runtime"

	"github.com/codefresh-io/cf-argo/pkg/kube"
)

var s Store

var (
	binaryName = "cf-argo"
	version    = "v99.99.99"
	gitCommit  = ""
	baseGitURL = "https://github.com/codefresh-io/argocd-template"
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
	KubeConfig *kube.Config
}

// Get returns the global store
func Get() *Store {
	return &s
}

func (s *Store) NewKubeClient(ctx context.Context) kube.Client {
	return kube.NewForConfig(ctx, s.KubeConfig)
}

func init() {
	s.BinaryName = binaryName
	s.BaseGitURL = baseGitURL
	s.KubeConfig = kube.NewConfig()

	initVersion()
}

func initVersion() {
	s.Version.Version = version
	s.Version.GitCommit = gitCommit
	s.Version.GoVersion = runtime.Version()
	s.Version.Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
