package helpers

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/codefresh-io/cf-argo/pkg/log"
	"github.com/yargevad/filepathx"
)

const (
	envNamePlaceholder = "envName"
)

func CopyDir(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		var relPath string = strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), info.Mode())
		} else {
			var data, err1 = ioutil.ReadFile(filepath.Join(source, relPath))
			if err1 != nil {
				return err1
			}
			return ioutil.WriteFile(filepath.Join(destination, relPath), data, info.Mode())
		}
	})
}

func RenderDirRecurse(pattern string, values interface{}) error {
	matches, err := filepathx.Glob(pattern)
	if err != nil {
		return err
	}

	for _, match := range matches {
		tpl, err := template.ParseFiles(match)
		if err != nil {
			return err
		}

		fw, err := os.OpenFile(match, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			return err
		}

		err = tpl.Execute(fw, values)
		fw.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// TODO maybe there is a more efficient way to do this
func RenameEnvNameRecurse(ctx context.Context, path, env string) error {
	matches, err := filepathx.Glob(filepath.Join(path, fmt.Sprintf("**/%s", envNamePlaceholder)))
	if err != nil {
		return err
	}

	// rename just directories
	for _, m := range matches {
		if strings.HasSuffix(m, envNamePlaceholder) {
			if err := renameEnvName(ctx, m, env); err != nil {
				return err
			}
		}
	}

	// run again to rename nested files with envName
	matches, err = filepathx.Glob(filepath.Join(path, fmt.Sprintf("**/%s.*", envNamePlaceholder)))
	if err != nil {
		return err
	}
	for _, m := range matches {
		if strings.Contains(m, envNamePlaceholder) {
			if err := renameEnvName(ctx, m, env); err != nil {
				return err
			}
		}
	}
	return nil
}

func renameEnvName(ctx context.Context, old, env string) error {
	ap, err := filepath.Abs(old)
	if err != nil {
		return err
	}
	newName := strings.Replace(ap, "envName", env, 1)
	log.G(ctx).WithFields(log.Fields{
		"old-path": ap,
		"new-path": newName,
	}).Debug("renaming with environment name")

	return os.Rename(ap, newName)
}
