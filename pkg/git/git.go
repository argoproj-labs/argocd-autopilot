package git

import (
	"context"
	"errors"
	"os"

	cferrors "github.com/codefresh-io/cf-argo/pkg/errors"
	"github.com/codefresh-io/cf-argo/pkg/log"

	"github.com/go-git/go-git/plumbing/transport"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type (
	// Repository represents a git repository
	Repository interface {
		Add(ctx context.Context, pattern string) error

		AddRemote(ctx context.Context, name, url string) error

		// Commits all files and returns the commit sha
		Commit(ctx context.Context, msg string) (string, error)

		Push(context.Context, *PushOptions) error
	}

	// Provider represents a git provider
	Provider interface {
		// CreateRepository creates the repository in the remote provider and returns a
		// clone url
		CreateRepository(ctx context.Context, opts *CreateRepositoryOptions) (string, error)
		Clone(ctx context.Context, opts *CloneOptions) (Repository, error)
	}

	// Options for a new git provider
	Options struct {
		Type string
		Auth *Auth
		Host string
	}

	// Auth for git provider
	Auth struct {
		Username string
		Password string
	}

	CloneOptions struct {
		// URL clone url
		URL string
		// Path where to clone to
		Path string
		Auth *Auth
		// Bare if true will not include .git directory
		Bare bool
	}

	PushOptions struct {
		RemoteName string
		Auth       *Auth
	}

	CreateRepositoryOptions struct {
		Owner   string
		Name    string
		Private bool
	}

	repo struct {
		r *gg.Repository
	}
)

// Errors
var (
	ErrProviderNotSupported = errors.New("git provider not supported")
)

// New creates a new git provider
func New(opts *Options) (Provider, error) {
	switch opts.Type {
	case "github":
		return newGithub(opts)
	default:
		return nil, ErrProviderNotSupported
	}
}

func Clone(ctx context.Context, opts *CloneOptions) (Repository, error) {
	if opts == nil {
		return nil, cferrors.ErrNilOpts
	}

	auth := getAuth(opts.Auth)
	log.G(ctx).WithFields(log.Fields{
		"url":  opts.URL,
		"path": opts.Path,
	}).Debug("cloning repo")

	cloneOpts := &gg.CloneOptions{
		Depth:    1,
		URL:      opts.URL,
		Auth:     auth,
		Progress: os.Stderr,
	}
	err := cloneOpts.Validate()
	if err != nil {
		return nil, err
	}

	r, err := gg.PlainCloneContext(ctx, opts.Path, opts.Bare, cloneOpts)
	if err != nil {
		return nil, err
	}

	return &repo{r}, nil
}

func Init(ctx context.Context, path string) (Repository, error) {
	if path == "" {
		path = "."
	}

	l := log.G(ctx).WithFields(log.Fields{
		"path": path,
	})

	l.Debug("initiallizing local repository")
	r, err := gg.PlainInit(path, false)
	if err != nil {
		return nil, err
	}
	l.Debug("local repository initiallized")

	return &repo{r}, err
}

func (r *repo) Add(ctx context.Context, pattern string) error {
	w, err := r.r.Worktree()
	if err != nil {
		return err
	}

	return w.AddGlob(pattern)
}

func (r *repo) AddRemote(ctx context.Context, name, url string) error {
	cfg := &config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	}
	err := cfg.Validate()
	if err != nil {
		return err
	}

	_, err = r.r.CreateRemote(cfg)
	if err != nil {
		return err
	}
	log.G(ctx).WithFields(log.Fields{
		"remote": cfg.Name,
	}).Debug("added new remote")

	return nil
}

func (r *repo) Commit(ctx context.Context, msg string) (string, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return "", err
	}

	h, err := wt.Commit(msg, &gg.CommitOptions{
		All: true,
	})
	if err != nil {
		return "", err
	}
	log.G(ctx).WithFields(log.Fields{
		"sha": h.String(),
	}).Debug("created new commit")

	return h.String(), err
}

func (r *repo) Push(ctx context.Context, opts *PushOptions) error {
	if opts == nil {
		return cferrors.ErrNilOpts
	}

	auth := getAuth(opts.Auth)
	pushOpts := &gg.PushOptions{
		RemoteName: opts.RemoteName,
		Auth:       auth,
		Progress:   os.Stdout,
	}
	err := pushOpts.Validate()
	if err != nil {
		return err
	}

	l := log.G(ctx).WithFields(log.Fields{
		"remote": pushOpts.RemoteName,
	})
	l.Debug("pushing to repo")

	err = r.r.PushContext(ctx, pushOpts)
	if err != nil {
		return err
	}

	l.Debug("pushed to repo")
	return nil
}

func getAuth(auth *Auth) transport.AuthMethod {
	if auth != nil {
		username := auth.Username
		if username == "" {
			username = "codefresh"
		}
		return &http.BasicAuth{
			Username: username,
			Password: auth.Password,
		}
	}

	return nil
}
