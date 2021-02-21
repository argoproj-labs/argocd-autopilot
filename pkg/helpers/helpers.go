package helpers

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/codefresh-io/cf-argo/pkg/log"
	"github.com/yargevad/filepathx"
)

const (
	envNamePlaceholder = "envName"
)

func ContextWithCancelOnSignals(ctx context.Context, sigs ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, sigs...)

	go func() {
		s := <-sig
		log.G(ctx).Debugf("got signal: %s", s)
		cancel()
	}()

	return ctx
}

func CopyDir(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		var relPath string = strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}

		absDst := filepath.Join(destination, relPath)
		if err = ensureDir(absDst); err != nil {
			return err
		}

		if info.IsDir() {
			err = os.Mkdir(absDst, info.Mode())
			if err != nil {
				if os.IsExist(err.(*os.PathError).Unwrap()) {
					return nil
				}
			}

			return err
		} else {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			return ioutil.WriteFile(absDst, data, info.Mode())
		}
	})
}

func ensureDir(path string) error {
	dstDir := filepath.Dir(path)
	if _, err := os.Stat(dstDir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		return os.MkdirAll(dstDir, 0755)
	}

	return nil
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

func RenameFilesWithEnvName(ctx context.Context, path, env string) error {
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
	matches, err = filepathx.Glob(filepath.Join(path, fmt.Sprintf("**/%s*.*", envNamePlaceholder)))
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

func ClearFolder(ctx context.Context, path string) error {
	err := removeContents(path)
	if err != nil {
		return err
	}

	_, err = os.Create(filepath.Join(path, "DUMMY"))
	if err != nil {
		return err
	}

	log.G(ctx).WithFields(log.Fields{
		"path": path,
	}).Debug("cleared folder")
	return nil
}

func removeContents(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return err
	}

	for _, file := range files {
		err = os.RemoveAll(file)
		if err != nil {
			return err
		}
	}

	return nil
}
