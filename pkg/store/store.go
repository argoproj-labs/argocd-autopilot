package store

import (
	"fmt"
	"runtime"
	"time"
)

var s Store

var (
	binaryName                       = "argocd-autopilot"
	version                          = "v99.99.99"
	buildDate                        = ""
	gitCommit                        = ""
	installationManifestsURL         = "manifests/base"
	installationManifestsInsecureURL = "manifests/insecure"
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
	BinaryName                       string
	Version                          Version
	InstallationManifestsURL         string
	InstallationManifestsInsecureURL string
}

var Default = struct {
	AppsDir              string
	ArgoCDName           string
	ArgoCDNamespace      string
	BaseDir              string
	BootsrtrapAppName    string
	BootsrtrapDir        string
	ClusterContextName   string
	ClusterResourcesDir  string
	DestServer           string
	DummyName            string
	DestServerAnnotation string
	GitHubUsername       string
	LabelKeyAppName      string
	LabelKeyAppManagedBy string
	LabelKeyAppPartOf    string
	LabelValueManagedBy  string
	OverlaysDir          string
	ProjectsDir          string
	RootAppName          string
	RepoCredsSecretName  string
	ArgoCDApplicationSet string
	WaitInterval         time.Duration
}{
	AppsDir:              "apps",
	ArgoCDName:           "argo-cd",
	ArgoCDNamespace:      "argocd",
	BaseDir:              "base",
	BootsrtrapAppName:    "autopilot-bootstrap",
	BootsrtrapDir:        "bootstrap",
	ClusterContextName:   "in-cluster",
	ClusterResourcesDir:  "cluster-resources",
	DestServer:           "https://kubernetes.default.svc",
	DestServerAnnotation: "argocd-autopilot.argoproj-labs.io/default-dest-server",
	DummyName:            "DUMMY",
	GitHubUsername:       "username",
	LabelKeyAppName:      "app.kubernetes.io/name",
	LabelKeyAppManagedBy: "app.kubernetes.io/managed-by",
	LabelKeyAppPartOf:    "app.kubernetes.io/part-of",
	LabelValueManagedBy:  "argocd-autopilot",
	OverlaysDir:          "overlays",
	ProjectsDir:          "projects",
	RootAppName:          "root",
	RepoCredsSecretName:  "autopilot-secret",
	ArgoCDApplicationSet: "argocd-applicationset",
	WaitInterval:         time.Second * 3,
}

// Get returns the global store
func Get() *Store {

	return &s
}

func init() {
	s.BinaryName = binaryName
	s.InstallationManifestsURL = installationManifestsURL
	s.InstallationManifestsInsecureURL = installationManifestsInsecureURL

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
