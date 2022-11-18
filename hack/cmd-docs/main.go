package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/argoproj-labs/argocd-autopilot/cmd/commands"

	"github.com/spf13/cobra/doc"
)

const (
	outputDir = "./docs/commands"
	home      = "/home/user"
)

var orgHome = os.Getenv("HOME")

func main() {
	log.Printf("org home: %s", orgHome)
	log.Printf("new home: %s", home)

	if err := doc.GenMarkdownTree(commands.NewRoot(), outputDir); err != nil {
		log.Fatal(err)
	}

	if err := replaceHome(); err != nil {
		log.Fatal(err)
	}
}

func replaceHome() error {
	files, err := fs.Glob(os.DirFS(outputDir), "*.md")
	if err != nil {
		return err
	}

	for _, fname := range files {
		fname = filepath.Join(outputDir, fname)
		data, err := os.ReadFile(fname)
		if err != nil {
			return err
		}

		datastr := string(data)
		newstr := strings.ReplaceAll(datastr, orgHome, home)

		if datastr == newstr {
			continue
		}

		log.Printf("replaced home at: %s", fname)

		err = os.WriteFile(fname, []byte(newstr), 0422)
		if err != nil {
			return err
		}
	}
	return nil
}
