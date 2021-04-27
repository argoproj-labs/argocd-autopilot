package main

import (
	"log"
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/argoproj/argocd-autopilot/cmd/commands"
)

var outputDir = "./docs/user-guide/commands"

func main() {
	// set HOME env var so that default values involve user's home directory do not depend on the running user.
	os.Setenv("HOME", "/home/user")

	err := doc.GenMarkdownTree(commands.NewAppCommand(), outputDir)
	if err != nil {
		log.Fatal(err)
	}

	err = doc.GenMarkdownTree(commands.NewRepoCommand(), outputDir)
	if err != nil {
		log.Fatal(err)
	}

	err = doc.GenMarkdownTree(commands.NewRepoBootstrapCommand(), outputDir)
	if err != nil {
		log.Fatal(err)
	}

	err = doc.GenMarkdownTree(commands.NewProjectCommand(), outputDir)
	if err != nil {
		log.Fatal(err)
	}
}
