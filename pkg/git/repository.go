package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git/gogit"
	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	billy "github.com/go-git/go-billy/v5"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:generate mockery --dir gogit --all --output gogit/mocks --case snake
//go:generate mockery --name Repository --filename repository.go

type (
	// Repository represents a git repository
	Repository interface {
		// Persist runs add, commit and push to the repository default remote
		Persist(ctx context.Context, opts *PushOptions) error
	}

	CloneOptions struct {
		AutoCreate bool
		Provider   string
		Repo       string
		Auth       Auth
		FS         fs.FS
		Progress   io.Writer
		url        string
		revision   string
		path       string
	}

	PushOptions struct {
		AddGlobPattern string
		CommitMsg      string
	}

	repo struct {
		gogit.Repository
		auth Auth
	}
)

// Errors
var (
	ErrNilOpts      = errors.New("options cannot be nil")
	ErrNoParse      = errors.New("must call Parse before using CloneOptions")
	ErrRepoNotFound = errors.New("git repository not found")
	ErrNoRemotes    = errors.New("no remotes in repository")
)

// go-git functions (we mock those in tests)
var (
	checkoutRef = func(r *repo, ref string) error {
		return r.checkoutRef(ref)
	}

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

func AddFlags(cmd *cobra.Command, bfs billy.Filesystem, prefix string) *CloneOptions {
	co := &CloneOptions{
		FS: fs.Create(bfs),
	}

	if prefix == "" {
		cmd.PersistentFlags().StringVarP(&co.Auth.Password, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
		cmd.PersistentFlags().StringVar(&co.Provider, "provider", "", fmt.Sprintf("The git provider, one of: %v", strings.Join(Providers(), "|")))
		cmd.PersistentFlags().StringVar(&co.Repo, "repo", "", "Repository URL [GIT_REPO]")

		util.Die(cmd.MarkPersistentFlagRequired("git-token"))
		util.Die(cmd.MarkPersistentFlagRequired("repo"))

		util.Die(viper.BindEnv("git-token", "GIT_TOKEN"))
		util.Die(viper.BindEnv("repo", "GIT_REPO"))
	} else {
		if !strings.HasSuffix(prefix, "-") {
			prefix += "-"
		}

		envPrefix := strings.ReplaceAll(strings.ToUpper(prefix), "-", "_")
		cmd.PersistentFlags().StringVar(&co.Auth.Password, prefix+"git-token", "", fmt.Sprintf("Your git provider api token [%sGIT_TOKEN]", envPrefix))
		cmd.PersistentFlags().StringVar(&co.Provider, prefix+"provider", "", fmt.Sprintf("The git provider, one of: %v", strings.Join(Providers(), "|")))
		cmd.PersistentFlags().StringVar(&co.Repo, prefix+"repo", "", fmt.Sprintf("Repository URL [%sGIT_REPO]", envPrefix))

		util.Die(viper.BindEnv(prefix+"git-token", envPrefix+"GIT_TOKEN"))
		util.Die(viper.BindEnv(prefix+"repo", envPrefix+"GIT_REPO"))
	}

	return co
}

func (o *CloneOptions) Parse() {
	var (
		host    string
		orgRepo string
		suffix  string
	)

	host, orgRepo, o.path, o.revision, _, suffix, _ = util.ParseGitUrl(o.Repo)
	o.url = host + orgRepo + suffix
}

func (o *CloneOptions) GetRepo(ctx context.Context) (Repository, fs.FS, error) {
	if o == nil {
		return nil, nil, ErrNilOpts
	}

	if o.url == "" {
		return nil, nil, ErrNoParse
	}

	r, err := clone(ctx, o)
	if err != nil {
		switch err {
		case transport.ErrRepositoryNotFound:
			if !o.AutoCreate {
				return nil, nil, err
			}

			log.G(ctx).Debug("no repository, creating a new one")
			_, err = createRepo(ctx, o)
			if err != nil {
				return nil, nil, err
			}

			fallthrough // a new repo will always start as empty - we need to init it locally
		case transport.ErrEmptyRemoteRepository:
			log.G(ctx).Debug("empty repository, initializing a new one with specified remote")
			r, err = initRepo(ctx, o)
			if err != nil {
				return nil, nil, err
			}
		default:
			return nil, nil, err
		}
	}

	bootstrapFS, err := o.FS.Chroot(o.path)
	if err != nil {
		return nil, nil, err
	}

	return r, fs.Create(bootstrapFS), nil
}

func (o *CloneOptions) URL() string {
	return o.url
}

func (o *CloneOptions) Revision() string {
	return o.revision
}

func (o *CloneOptions) Path() string {
	return o.path
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
		Progress: os.Stderr,
	})
}

var clone = func(ctx context.Context, opts *CloneOptions) (*repo, error) {
	if opts == nil {
		return nil, ErrNilOpts
	}

	if opts.Progress == nil {
		opts.Progress = os.Stderr
	}

	cloneOpts := &gg.CloneOptions{
		URL:      opts.url,
		Auth:     getAuth(opts.Auth),
		Depth:    1,
		Progress: opts.Progress,
	}

	log.G(ctx).WithField("url", opts.url).Debug("cloning git repo")
	r, err := ggClone(ctx, memory.NewStorage(), opts.FS, cloneOpts)
	if err != nil {
		return nil, err
	}

	repo := &repo{Repository: r, auth: opts.Auth}

	if opts.revision != "" {
		if err := checkoutRef(repo, opts.revision); err != nil {
			return nil, err
		}
	}

	return repo, nil
}

var createRepo = func(ctx context.Context, opts *CloneOptions) (string, error) {
	host, orgRepo, _, _, _, _, _ := util.ParseGitUrl(opts.Repo)
	providerType := opts.Provider
	if providerType == "" {
		u, err := url.Parse(host)
		if err != nil {
			return "", err
		}

		providerType = u.Hostname()
	}

	p, err := NewProvider(&ProviderOptions{
		Type: providerType,
		Auth: &opts.Auth,
		Host: host,
	})
	if err != nil {
		return "", fmt.Errorf("Failed to create the repository: %w\nYou can try to manually create it before trying again.", err)
	}

	s := strings.Split(orgRepo, "/")
	if len(s) < 2 {
		return "", fmt.Errorf("Failed parsing organization and repo from '%s'", orgRepo)
	}

	owner := strings.Join(s[:len(s)-1], "/")
	name := s[len(s)-1]
	return p.CreateRepository(ctx, &CreateRepoOptions{
		Owner:   owner,
		Name:    name,
		Private: true,
	})
}

var initRepo = func(ctx context.Context, opts *CloneOptions) (*repo, error) {
	ggr, err := ggInitRepo(memory.NewStorage(), opts.FS)
	if err != nil {
		return nil, err
	}

	r := &repo{Repository: ggr, auth: opts.Auth}
	if err = r.addRemote("origin", opts.url); err != nil {
		return nil, err
	}

	return r, r.initBranch(ctx, opts.revision)
}

func (r *repo) checkoutRef(ref string) error {
	hash, err := r.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		if err != plumbing.ErrReferenceNotFound {
			return err
		}

		log.G().WithField("ref", ref).Debug("failed resolving ref, trying to resolve from remote branch")
		remotes, err := r.Remotes()
		if err != nil {
			return err
		}

		if len(remotes) == 0 {
			return ErrNoRemotes
		}

		remoteref := fmt.Sprintf("%s/%s", remotes[0].Config().Name, ref)
		hash, err = r.ResolveRevision(plumbing.Revision(remoteref))
		if err != nil {
			return err
		}
	}

	wt, err := worktree(r)
	if err != nil {
		return err
	}

	log.G().WithFields(log.Fields{
		"ref":  ref,
		"hash": hash.String(),
	}).Debug("checking out commit")
	return wt.Checkout(&gg.CheckoutOptions{
		Hash: *hash,
	})
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

func getAuth(auth Auth) transport.AuthMethod {
	if auth.Password == "" {
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
