package fs

import (
	"fmt"
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/util"
	"github.com/go-git/go-billy/v5"
)

//go:generate mockery -name FS -filename fs.go
//go:generate mockery -name File -filename file.go
type FS interface {
	billy.Filesystem

	CheckExistsOrWrite(path string, data []byte) (bool, error)

	// Exists checks if the provided path exists in the provided filesystem.
	Exists(path string) (bool, error)

	// ExistsOrDie checks if the provided path exists in the provided filesystem, or panics on any error other then ErrNotExist
	ExistsOrDie(path string) bool
	WriteFile(path string, data []byte) (int, error)
}

type fsimpl struct {
	billy.Filesystem
}

type File interface {
	billy.File
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

func (fs *fsimpl) ExistsOrDie(path string) bool {
	exists, err := fs.Exists(path)
	util.Die(err)
	return exists
}

func (fs *fsimpl) WriteFile(path string, data []byte) (int, error) {
	f, err := fs.Create(path) // recursively creates nested dirs if needs to
	if err != nil {
		return 0, err
	}

	return f.Write(data)
}
