package main

import (
	"log"
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/argoproj/argocd-autopilot/cmd/commands"
)

var outputDir = "./docs/commands"

func main() {
	// set HOME env var so that default values involve user's home directory do not depend on the running user.
	os.Setenv("HOME", "/home/user")

	err := doc.GenMarkdownTree(commands.NewRoot(), outputDir)
	if err != nil {
		log.Fatal(err)
	}
}
