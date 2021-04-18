package fs

import (
	"fmt"
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/store"
	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/go-git/go-billy/v5"
)

//go:generate mockery -name FS -filename fs.go

type FS interface {
	billy.Filesystem

	CheckExistsOrWrite(path string, data []byte) (bool, error)
	MustChroot(newRoot string)
	Exists(path string) (bool, error)
	MustCheckEnvExists(envName string) bool
	MustExists(path string, notExistsMsg ...string)
	WriteFile(path string, data []byte) (int, error)
}

type fsimpl struct {
	billy.Filesystem
}

func Create(bfs billy.Filesystem) FS {
	return &fsimpl{bfs}
}

func (fs *fsimpl) CheckExistsOrWrite(path string, data []byte) (bool, error) {
	exists, err := fs.Exists(path)
	if err != nil {
		return false, fmt.Errorf("failed to check if file exists on repo: %s: %w", path, err)
	}

	if exists {
		return true, nil
	}

	if _, err = fs.WriteFile(path, data); err != nil {
		return false, fmt.Errorf("failed to create file at: %s: %w", path, err)
	}

	return false, nil
}

func (fs *fsimpl) Exists(path string) (bool, error) {
	if _, err := fs.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}

		return false, nil
	}

	return true, nil
}

func (fs *fsimpl) MustExists(path string, notExistsMsg ...string) {
	exists, err := fs.Exists(path)
	util.Die(err)

	if !exists {
		util.Die(fmt.Errorf("path does not exist: %s", path), notExistsMsg...)
	}
}

func (fs *fsimpl) MustCheckEnvExists(envName string) bool {
	ok, err := fs.Exists(fs.Join(store.Default.EnvsDir, fmt.Sprintf("%s.yaml", envName)))
	if err != nil {
		util.Die(err)
	}

	return ok
}

func (fs *fsimpl) MustChroot(newRoot string) {
	var err error
	fs.Filesystem, err = fs.Chroot(newRoot)
	util.Die(err, "failed to chroot")
}

func (fs *fsimpl) WriteFile(path string, data []byte) (int, error) {
	f, err := fs.Create(path) // recursively creates nested dirs if needs to
	if err != nil {
		return 0, err
	}

	return f.Write(data)
}
