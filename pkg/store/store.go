package store

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var s Store

var (
	binaryName = ""
	version    = "v99.99.99"
	buildDate  = ""
	gitCommit  = ""
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

func (s *Store) NewVersionCommand() *cobra.Command {
	var opts struct {
		long bool
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "show cli version",
		Run: func(cmd *cobra.Command, args []string) {
			if opts.long {
				fmt.Printf("Version: %s\n", s.Version.Version)
				fmt.Printf("BuildDate: %s\n", s.Version.BuildDate)
				fmt.Printf("GitCommit: %s\n", s.Version.GitCommit)
				fmt.Printf("GoVersion: %s\n", s.Version.GoVersion)
				fmt.Printf("GoCompiler: %s\n", s.Version.GoCompiler)
				fmt.Printf("Platform: %s\n", s.Version.Platform)
			} else {
				fmt.Printf("%+s\n", s.Version.Version)
			}
		},
	}

	cmd.Flags().BoolVar(&opts.long, "long", false, "display full version information")

	return cmd
}

func initVersion() {
	s.Version.Version = version
	s.Version.BuildDate = buildDate
	s.Version.GitCommit = gitCommit
	s.Version.GoVersion = runtime.Version()
	s.Version.GoCompiler = runtime.Compiler
	s.Version.Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
