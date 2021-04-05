package git

import (
	"context"
	"errors"
	"net/url"
	"os"

	"github.com/go-git/go-billy/v5"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

type (
	// Repository represents a git repository
	Repository interface {
		Add(ctx context.Context, pattern string) error

		AddRemote(ctx context.Context, name, url string) error

		// Commits all files and returns the commit sha
		Commit(ctx context.Context, msg string) (string, error)

		Push(context.Context, *PushOptions) error

		IsNewRepo() (bool, error)

		Root() (string, error)
	}

	CloneOptions struct {
		// URL clone url
		URL      string
		Revision string
		Auth     *Auth
	}

	PushOptions struct {
		Auth *Auth
	}

	repo struct {
		r *gg.Repository
	}
)

// Errors
var (
	ErrNilOpts      = errors.New("options cannot be nil")
	ErrRepoNotFound = errors.New("git repository not found")
)

// go-git functions (we mock those in tests)
var (
	clone    = gg.CloneContext
	initRepo = gg.Init
)

func Clone(ctx context.Context, fs billy.Filesystem, opts *CloneOptions) (Repository, error) {
	if opts == nil {
		return nil, ErrNilOpts
	}

	auth := getAuth(opts.Auth)

	cloneOpts := &gg.CloneOptions{
		URL:          opts.URL,
		Auth:         auth,
		SingleBranch: true,
		Depth:        1,
		Progress:     os.Stderr,
		Tags:         gg.NoTags,
	}

	if opts.Revision != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(opts.Revision)
	}

	err := cloneOpts.Validate()
	if err != nil {
		return nil, err
	}

	r, err := clone(ctx, memory.NewStorage(), fs, cloneOpts)
	if err != nil {
		return nil, err
	}

	return &repo{r}, nil
}

func Init(fs billy.Filesystem) (Repository, error) {
	r, err := initRepo(memory.NewStorage(), fs)
	if err != nil {
		return nil, err
	}

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

	return h.String(), err
}

func (r *repo) Push(ctx context.Context, opts *PushOptions) error {
	if opts == nil {
		return ErrNilOpts
	}

	auth := getAuth(opts.Auth)
	pushOpts := &gg.PushOptions{
		Auth:     auth,
		Progress: os.Stdout,
	}

	err := pushOpts.Validate()
	if err != nil {
		return err
	}

	err = r.r.PushContext(ctx, pushOpts)
	if err != nil {
		return err
	}

	return nil
}

func (r *repo) IsNewRepo() (bool, error) {
	remotes, err := r.r.Remotes()
	if err != nil {
		return false, err
	}

	return len(remotes) == 0, nil
}

func (r *repo) Root() (string, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return "", err
	}

	return wt.Filesystem.Root(), nil
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

func getRef(cloneURL string) string {
	u, err := url.Parse(cloneURL)
	if err != nil {
		return ""
	}

	return u.Fragment
}
