package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"
	"github.com/argoproj-labs/argocd-autopilot/pkg/git/gogit"
	"github.com/argoproj-labs/argocd-autopilot/pkg/log"
	"github.com/argoproj-labs/argocd-autopilot/pkg/store"
	"github.com/argoproj-labs/argocd-autopilot/pkg/util"

	billy "github.com/go-git/go-billy/v5"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:generate mockgen -destination=./mocks/repository.go -package=mocks -source=./repository.go Repository
//go:generate mockgen -destination=./gogit/mocks/repository.go -package=mocks -source=./gogit/repo.go Repository
//go:generate mockgen -destination=./gogit/mocks/worktree.go -package=mocks -source=./gogit/worktree.go Worktree

type (
	// Repository represents a git repository
	Repository interface {
		// Persist runs add, commit and push to the repository default remote
		Persist(ctx context.Context, opts *PushOptions) (string, error)
		// CurrentBranch returns the name of the current branch
		CurrentBranch() (string, error)
	}

	AddFlagsOptions struct {
		FS               billy.Filesystem
		Prefix           string
		CreateIfNotExist bool
		// CloneForWrite if true will not allow 'ref' query param which is not
		// a branch name
		CloneForWrite bool
		Optional      bool
	}

	CloneOptions struct {
		Provider         string
		Repo             string
		Auth             Auth
		FS               fs.FS
		Progress         io.Writer
		CreateIfNotExist bool
		CloneForWrite    bool
		UpsertBranch     bool
	
		url              string
		revision         string
		path             string
		provider         Provider
	}

	PushOptions struct {
		Provider       string
		AddGlobPattern string
		CommitMsg      string
		Progress       io.Writer
	}

	repo struct {
		gogit.Repository
		auth     Auth
		progress io.Writer
		provider Provider
	}
)

// Errors
var (
	ErrNilOpts      = errors.New("options cannot be nil")
	ErrNoParse      = errors.New("must call Parse before using CloneOptions")
	ErrRepoNotFound = errors.New("git repository not found")
	ErrNoRemotes    = errors.New("no remotes in repository")
)

// Defaults
const (
	pushRetries        = 3
	failureBackoffTime = 3 * time.Second
)

// go-git functions (we mock those in tests)
var (
	checkoutRef = func(r *repo, ref string) error {
		return r.checkoutRef(ref)
	}

	checkoutBranch = func(r *repo, branch string, upsertBranch bool) error {
		return r.checkoutBranch(branch, upsertBranch)
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

	defaultBranch = func() (string, error) {
		cfg, err := config.LoadConfig(config.GlobalScope)
		if err != nil {
			return "", fmt.Errorf("failed to load global git config: %w", err)
		}

		if cfg.Init.DefaultBranch == "" {
			return "main", nil
		}

		return cfg.Init.DefaultBranch, nil
	}
)

func AddFlags(cmd *cobra.Command, opts *AddFlagsOptions) *CloneOptions {
	co := &CloneOptions{
		FS:               fs.Create(opts.FS),
		CreateIfNotExist: opts.CreateIfNotExist,
		CloneForWrite:    opts.CloneForWrite,
	}

	if opts.Prefix != "" && !strings.HasSuffix(opts.Prefix, "-") {
		opts.Prefix += "-"
	}

	envPrefix := strings.ReplaceAll(strings.ToUpper(opts.Prefix), "-", "_")
	cmd.PersistentFlags().StringVar(&co.Auth.Password, opts.Prefix+"git-token", "", fmt.Sprintf("Your git provider api token [%sGIT_TOKEN]", envPrefix))
	cmd.PersistentFlags().StringVar(&co.Auth.Username, opts.Prefix+"git-user", "", fmt.Sprintf("Your git provider user name [%sGIT_USER] (not required in GitHub)", envPrefix))
	cmd.PersistentFlags().StringVar(&co.Repo, opts.Prefix+"repo", "", fmt.Sprintf("Repository URL [%sGIT_REPO]", envPrefix))

	util.Die(viper.BindEnv(opts.Prefix+"git-token", envPrefix+"GIT_TOKEN"))
	util.Die(viper.BindEnv(opts.Prefix+"git-user", envPrefix+"GIT_USER"))
	util.Die(viper.BindEnv(opts.Prefix+"repo", envPrefix+"GIT_REPO"))

	if opts.Prefix == "" {
		cmd.Flag("git-token").Shorthand = "t"
		cmd.Flag("git-user").Shorthand = "u"
	}

	if opts.CreateIfNotExist {
		cmd.PersistentFlags().StringVar(&co.Provider, opts.Prefix+"provider", "", fmt.Sprintf("The git provider, one of: %v", strings.Join(Providers(), "|")))
	}

	if opts.CloneForWrite {
		cmd.PersistentFlags().BoolVarP(&co.UpsertBranch, opts.Prefix+"upsert-branch", "b", false, "If true will try to checkout the specified branch and create it if it doesn't exist")
	}

	if !opts.Optional {
		util.Die(cmd.MarkPersistentFlagRequired(opts.Prefix + "git-token"))
		util.Die(cmd.MarkPersistentFlagRequired(opts.Prefix + "repo"))
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

	if o.Auth.Username == "" {
		o.Auth.Username = store.Default.GitHubUsername
	}
}

func (o *CloneOptions) GetRepo(ctx context.Context) (Repository, fs.FS, error) {
	if o == nil {
		return nil, nil, ErrNilOpts
	}

	if o.url == "" {
		return nil, nil, ErrNoParse
	}

	provider, err := getProvider(o.Provider, o.url, &o.Auth)
	if err != nil {
		return nil, nil, err
	}

	o.provider = provider

	r, err := clone(ctx, o)
	if err != nil {
		switch err {
		case transport.ErrRepositoryNotFound:
			if !o.CreateIfNotExist {
				return nil, nil, err
			}

			log.G(ctx).Infof("repository '%s' was not found, trying to create it...", o.Repo)
			_, err = createRepo(ctx, o)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create repository: %w", err)
			}

			fallthrough // a new repo will always start as empty - we need to init it locally
		case transport.ErrEmptyRemoteRepository:
			log.G(ctx).Info("empty repository, initializing a new one with specified remote")
			r, err = initRepo(ctx, o)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to initialize repository: %w", err)
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

func (r *repo) Persist(ctx context.Context, opts *PushOptions) (string, error) {
	if opts == nil {
		return "", ErrNilOpts
	}

	progress := opts.Progress
	if progress == nil {
		progress = r.progress
	}

	h, err := r.commit(ctx, opts)
	if err != nil {
		return "", err
	}

	for try := 0; try < pushRetries; try++ {
		err = r.PushContext(ctx, &gg.PushOptions{
			Auth:     getAuth(r.auth),
			Progress: progress,
		})
		if err == nil || !errors.Is(err, transport.ErrRepositoryNotFound) {
			break
		}

		log.G(ctx).WithFields(log.Fields{
			"retry": try,
			"err":   err.Error(),
		}).Warn("Failed to push to repository, trying again in 3 seconds...")

		time.Sleep(failureBackoffTime)
	}

	return h.String(), err
}

func (r *repo) CurrentBranch() (string, error) {
	ref, err := r.Head()
	if err != nil {
		return "", fmt.Errorf("failed to resolve ref: %w", err)
	}

	return ref.Name().Short(), nil
}

func (r *repo) commit(ctx context.Context, opts *PushOptions) (*plumbing.Hash, error) {
	var h plumbing.Hash

	author, err := r.getAuthor(ctx)
	if err != nil {
		return nil, err
	}

	w, err := worktree(r)
	if err != nil {
		return nil, err
	}

	addPattern := "."
	if opts.AddGlobPattern != "" {
		addPattern = opts.AddGlobPattern
	}

	if err := w.AddGlob(addPattern); err != nil {
		// allowing the glob pattern to not match any files, in case of add-all ("."), like with initBranch for example
		if addPattern != "." || err != gg.ErrGlobNoMatches {
			return nil, err
		}
	}

	h, err = w.Commit(opts.CommitMsg, &gg.CommitOptions{
		All:    true,
		Author: author,
	})
	if err != nil {
		return nil, err
	}

	return &h, nil
}

func (r *repo) getAuthor(ctx context.Context) (*object.Signature, error) {
	cfg, err := r.ConfigScoped(config.SystemScope)
	if err != nil {
		return nil, fmt.Errorf("failed to get gitconfig: %w", err)
	}

	username := cfg.User.Name
	email := cfg.User.Email

	if username == "" || email == "" {
		username, email, err = r.provider.GetAuthor(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get author information: %w", err)
		}

		if username == "" || email == "" {
			return nil, fmt.Errorf("missing required author information in git config, make sure your git config contains a 'user.name' and 'user.email'")
		}
	}

	return &object.Signature{
		Name:  username,
		Email: email,
		When:  time.Now(),
	}, nil
}

var clone = func(ctx context.Context, opts *CloneOptions) (*repo, error) {
	var (
		err            error
		r              gogit.Repository
		curPushRetries = pushRetries
	)

	if opts == nil {
		return nil, ErrNilOpts
	}

	progress := opts.Progress
	if progress == nil {
		progress = os.Stderr
	}

	cloneOpts := &gg.CloneOptions{
		URL:      opts.url,
		Auth:     getAuth(opts.Auth),
		Depth:    1,
		Progress: progress,
	}

	log.G(ctx).WithField("url", opts.url).Debug("cloning git repo")

	if opts.CreateIfNotExist {
		curPushRetries = 1 // no retries
	}

	for try := 0; try < curPushRetries; try++ {
		r, err = ggClone(ctx, memory.NewStorage(), opts.FS, cloneOpts)
		if err == nil || !errors.Is(err, transport.ErrRepositoryNotFound) {
			break
		}

		log.G(ctx).WithFields(log.Fields{
			"retry": try,
			"err":   err.Error(),
		}).Debug("Failed to clone repository, trying again in 3 seconds...")

		time.Sleep(failureBackoffTime)
	}

	if err != nil {
		return nil, err
	}

	repo := &repo{
		Repository: r,
		auth:       opts.Auth,
		progress:   progress,
		provider:   opts.provider,
	}

	if opts.revision != "" {
		if opts.CloneForWrite {
			log.G(ctx).WithFields(log.Fields{
				"branch": opts.revision,
				"upsert": opts.UpsertBranch,
			}).Debug("Trying to checkout branch")

			if err := checkoutBranch(repo, opts.revision, opts.UpsertBranch); err != nil {
				return nil, err
			}
		} else {
			log.G(ctx).WithField("ref", opts.revision).Debug("Trying to checkout ref")

			if err := checkoutRef(repo, opts.revision); err != nil {
				return nil, err
			}
		}
	}

	return repo, nil
}

var createRepo = func(ctx context.Context, opts *CloneOptions) (string, error) {
	_, orgRepo, _, _, _, _, _ := util.ParseGitUrl(opts.Repo)
	return opts.provider.CreateRepository(ctx, orgRepo)
}

func getProvider(providerType, repoUrl string, auth *Auth) (Provider, error) {
	if providerType == "" {
		u, err := url.Parse(repoUrl)
		if err != nil {
			return nil, err
		}

		if strings.Contains(u.Hostname(), AzureHostName) {
			providerType = Azure
		} else {
			providerType = strings.TrimSuffix(u.Hostname(), ".com")
		}

		log.G().Warnf("--provider not specified, assuming provider from url: %s", providerType)
	}

	return newProvider(&ProviderOptions{
		Type: providerType,
		Auth: auth,
		Host: repoUrl,
	})
}

func getDefaultRepoOptions(orgRepo string) (*CreateRepoOptions, error) {
	s := strings.Split(orgRepo, "/")
	if len(s) < 2 {
		return nil, fmt.Errorf("failed parsing organization and repo from '%s'", orgRepo)
	}

	owner := strings.Join(s[:len(s)-1], "/")
	name := s[len(s)-1]
	return &CreateRepoOptions{
		Owner:   owner,
		Name:    name,
		Private: true,
	}, nil
}

var initRepo = func(ctx context.Context, opts *CloneOptions) (*repo, error) {
	ggr, err := ggInitRepo(memory.NewStorage(), opts.FS)
	if err != nil {
		return nil, err
	}

	progress := opts.Progress
	if progress == nil {
		progress = os.Stderr
	}

	r := &repo{
		Repository: ggr,
		auth:       opts.Auth,
		progress:   progress,
		provider:   opts.provider,
	}
	if err = r.addRemote("origin", opts.url); err != nil {
		return nil, err
	}

	return r, r.initBranch(ctx, opts.revision)
}

func (r *repo) checkoutBranch(branch string, upsertBranch bool) error {
	wt, err := worktree(r)
	if err != nil {
		return err
	}

	err = wt.Checkout(&gg.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
	if err != plumbing.ErrReferenceNotFound {
		return err
	}

	remotes, err := r.Remotes()
	if err != nil {
		return err
	}

	if len(remotes) == 0 {
		return ErrNoRemotes
	}

	err = wt.Checkout(&gg.CheckoutOptions{
		Branch: plumbing.NewRemoteReferenceName(remotes[0].Config().Name, branch),
	})
	if err != nil {
		if err == plumbing.ErrReferenceNotFound && upsertBranch {
			// no remote branch but create is true
			// so we will create a new local branch
			return wt.Checkout(&gg.CheckoutOptions{
				Branch: plumbing.NewBranchReferenceName(branch),
				Create: true,
			})

		}

		return err
	}

	// if succeeded to checkout to a remote branch with this name,
	// checkout to a local branch from the remote branch
	return wt.Checkout(&gg.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
		Create: true,
	})
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
	_, err := r.commit(ctx, &PushOptions{
		CommitMsg: "initial commit",
	})

	if err != nil {
		return fmt.Errorf("failed to commit while trying to initialize the branch. Error: %w", err)
	}

	if branchName == "" {
		branchName, err = defaultBranch()
		if err != nil {
			return err
		}
	}

	b := plumbing.NewBranchReferenceName(branchName)
	log.G(ctx).WithField("branch", b).Debug("checking out branch")

	w, err := worktree(r)
	if err != nil {
		return err
	}

	return w.Checkout(&gg.CheckoutOptions{
		Branch: b,
		Create: true,
	})
}

func getAuth(auth Auth) transport.AuthMethod {
	if auth.Password == "" {
		return nil
	}

	return &http.BasicAuth{
		Username: auth.Username,
		Password: auth.Password,
	}
}
