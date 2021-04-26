package main

import (
	"log"
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/argoproj/argocd-autopilot/cmd/commands"
)

func main() {
	// set HOME env var so that default values involve user's home directory do not depend on the running user.
	os.Setenv("HOME", "/home/user")

	err := doc.GenMarkdownTree(commands.NewAppCommand(), "./docs/user-guide/commands")
	if err != nil {
		log.Fatal(err)
	}

	err = doc.GenMarkdownTree(commands.NewRepoCommand(), "./docs/user-guide/commands")
	if err != nil {
		log.Fatal(err)
	}

	err = doc.GenMarkdownTree(commands.NewRepoBootstrapCommand(), "./docs/user-guide/commands")
	if err != nil {
		log.Fatal(err)
	}

	err = doc.GenMarkdownTree(commands.NewProjectCommand(), "./docs/user-guide/commands")
	if err != nil {
		log.Fatal(err)
	}
}