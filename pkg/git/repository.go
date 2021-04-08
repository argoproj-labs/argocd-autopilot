package git

import (
	"context"
	"errors"
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/log"
	billy "github.com/go-git/go-billy/v5"
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
		// Persist runs add, commit and push to the repository default remote
		Persist(ctx context.Context, opts *PushOptions) error
	}

	CloneOptions struct {
		// URL clone url
		URL      string
		Revision string
		Auth     *Auth
		FS       billy.Filesystem
	}

	PushOptions struct {
		AddGlobPattern string
		CommitMsg      string
	}

	repo struct {
		*gg.Repository
		auth *Auth
	}
)

// Errors
var (
	ErrNilOpts      = errors.New("options cannot be nil")
	ErrRepoNotFound = errors.New("git repository not found")
)

// go-git functions (we mock those in tests)
var (
	ggClone    = gg.CloneContext
	ggInitRepo = gg.Init
)

func Clone(ctx context.Context, opts *CloneOptions) (Repository, error) {
	r, err := clone(ctx, opts.FS, &CloneOptions{
		URL:      opts.URL,
		Revision: opts.Revision,
		Auth:     opts.Auth,
	})
	if err != nil {
		if err == transport.ErrEmptyRemoteRepository {
			log.G(ctx).Debug("empty repository, initializing new one with specified remote")
			return initRepo(ctx, opts)
		}
		return nil, err
	}

	return r, nil
}

func clone(ctx context.Context, fs billy.Filesystem, opts *CloneOptions) (*repo, error) {
	if opts == nil {
		return nil, ErrNilOpts
	}

	cloneOpts := &gg.CloneOptions{
		URL:          opts.URL,
		Auth:         getAuth(opts.Auth),
		SingleBranch: true,
		Depth:        1,
		Progress:     os.Stderr,
		Tags:         gg.NoTags,
	}

	if opts.Revision != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(opts.Revision)
	}

	log.G(ctx).WithFields(log.Fields{
		"url": opts.URL,
		"rev": opts.Revision,
	}).Debug("cloning git repo")
	r, err := ggClone(ctx, memory.NewStorage(), fs, cloneOpts)
	if err != nil {
		return nil, err
	}

	return &repo{Repository: r, auth: opts.Auth}, nil
}

func (r *repo) Persist(ctx context.Context, opts *PushOptions) error {
	if opts == nil {
		return ErrNilOpts
	}
	addPattern := "."

	if opts.AddGlobPattern != "" {
		addPattern = opts.AddGlobPattern
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	if err := w.AddGlob(addPattern); err != nil {
		return err
	}

	if _, err = r.commit(ctx, opts.CommitMsg); err != nil {
		return err
	}

	return r.PushContext(ctx, &gg.PushOptions{
		Auth:     getAuth(r.auth),
		Progress: os.Stdout,
	})
}

func initRepo(ctx context.Context, opts *CloneOptions) (Repository, error) {
	ggr, err := ggInitRepo(memory.NewStorage(), opts.FS)
	if err != nil {
		return nil, err
	}

	r := &repo{Repository: ggr, auth: opts.Auth}
	if err = r.addRemote(ctx, "origin", opts.URL); err != nil {
		return nil, err
	}

	return r, r.checkout(ctx, opts.Revision)
}

func (r *repo) addRemote(ctx context.Context, name, url string) error {
	cfg := &config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	}

	_, err := r.CreateRemote(cfg)
	if err != nil {
		return err
	}

	return nil
}

func (r *repo) checkout(ctx context.Context, branchName string) error {
	wt, err := r.Worktree()
	if err != nil {
		return err
	}

	_, err = wt.Commit("initial commit", &gg.CommitOptions{})
	if err != nil {
		return err
	}

	log.G(ctx).WithField("branch", branchName).Debug("checking out branch")
	b := plumbing.NewBranchReferenceName(branchName)
	return wt.Checkout(&gg.CheckoutOptions{
		Branch: b,
		Create: true,
	})
}

func (r *repo) commit(ctx context.Context, msg string) (string, error) {
	wt, err := r.Worktree()
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

func getAuth(auth *Auth) transport.AuthMethod {
	if auth != nil {
		username := auth.Username
		if username == "" {
			username = "git"
		}

		return &http.BasicAuth{
			Username: username,
			Password: auth.Password,
		}
	}

	return nil
}
