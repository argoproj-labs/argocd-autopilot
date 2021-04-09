package git

import (
	"context"
	"errors"
	"os"

	"github.com/argoproj/argocd-autopilot/pkg/git/gogit"
	"github.com/argoproj/argocd-autopilot/pkg/log"
	billy "github.com/go-git/go-billy/v5"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/memory"
)

//go:generate interfacer -for github.com/go-git/go-git/v5.Repository -as gogit.Repository -o gogit/repo.go
//go:generate interfacer -for github.com/go-git/go-git/v5.Worktree -as gogit.Worktree -o gogit/worktree.go
//go:generate mockery -dir gogit -all -output gogit/mocks -case snake

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
		gogit.Repository
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
	ggClone = func(ctx context.Context, s storage.Storer, worktree billy.Filesystem, o *gg.CloneOptions) (gogit.Repository, error) {
		return gg.CloneContext(ctx, s, worktree, o)
	}

	ggInitRepo = func(s storage.Storer, worktree billy.Filesystem) (gogit.Repository, error) {
		return gg.Init(s, worktree)
	}

	worktree = func(r gogit.Repository) (gogit.Worktree, error) {
		return r.Worktree()
	}
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

func (r *repo) Persist(ctx context.Context, opts *PushOptions) error {
	if opts == nil {
		return ErrNilOpts
	}
	addPattern := "."

	if opts.AddGlobPattern != "" {
		addPattern = opts.AddGlobPattern
	}

	w, err := worktree(r)
	if err != nil {
		return err
	}

	if err := w.AddGlob(addPattern); err != nil {
		return err
	}

	if _, err = w.Commit(opts.CommitMsg, &gg.CommitOptions{All: true}); err != nil {
		return err
	}

	return r.PushContext(ctx, &gg.PushOptions{
		Auth:     getAuth(r.auth),
		Progress: os.Stdout,
	})
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

func initRepo(ctx context.Context, opts *CloneOptions) (Repository, error) {
	ggr, err := ggInitRepo(memory.NewStorage(), opts.FS)
	if err != nil {
		return nil, err
	}

	r := &repo{Repository: ggr, auth: opts.Auth}
	if err = r.addRemote("origin", opts.URL); err != nil {
		return nil, err
	}

	return r, r.initBranch(ctx, opts.Revision)
}

func (r *repo) addRemote(name, url string) error {
	_, err := r.CreateRemote(&config.RemoteConfig{Name: name, URLs: []string{url}})
	return err
}

func (r *repo) initBranch(ctx context.Context, branchName string) error {
	w, err := worktree(r)
	if err != nil {
		return err
	}

	_, err = w.Commit("initial commit", &gg.CommitOptions{})
	if err != nil {
		return err
	}

	if branchName == "" {
		return nil
	}

	b := plumbing.NewBranchReferenceName(branchName)
	log.G(ctx).WithField("branch", b).Debug("checking out branch")
	return w.Checkout(&gg.CheckoutOptions{
		Branch: b,
		Create: true,
	})
}

func getAuth(auth *Auth) transport.AuthMethod {
	if auth == nil {
		return nil
	}

	username := auth.Username
	if username == "" {
		username = "git"
	}

	return &http.BasicAuth{
		Username: username,
		Password: auth.Password,
	}
}
