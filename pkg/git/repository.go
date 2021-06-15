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
		// URL clone url
		Repo     string
		Auth     Auth
		FS       fs.FS
		Progress io.Writer
		url      string
		revision string
		path     string
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

const (
	gitSuffix    = ".git"
	gitDelimiter = "_git/"
)

// Errors
var (
	ErrNilOpts      = errors.New("options cannot be nil")
	ErrNoParse      = errors.New("must call Parse before using CloneOptions")
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

func AddFlags(cmd *cobra.Command, bfs billy.Filesystem, prefix string) *CloneOptions {
	co := &CloneOptions{
		FS: fs.Create(bfs),
	}

	if prefix == "" {
		cmd.PersistentFlags().StringVarP(&co.Auth.Password, "git-token", "t", "", "Your git provider api token [GIT_TOKEN]")
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

	host, orgRepo, o.path, o.revision, suffix = parseGitUrl(o.Repo)
	o.url = host + orgRepo + suffix
}

func (o *CloneOptions) Clone(ctx context.Context) (Repository, fs.FS, error) {
	if o == nil {
		return nil, nil, ErrNilOpts
	}

	if o.url == "" {
		return nil, nil, ErrNoParse
	}

	r, err := clone(ctx, o)
	if err != nil {
		if err == transport.ErrEmptyRemoteRepository {
			log.G(ctx).Debug("empty repository, initializing new one with specified remote")
			r, err = initRepo(ctx, o)
		}
	}

	if err != nil {
		return nil, nil, err
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
	return plumbing.ReferenceName(o.revision).Short()
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
		Tags:     gg.NoTags,
	}

	if opts.revision != "" {
		cloneOpts.ReferenceName = plumbing.ReferenceName(opts.revision)
	}

	log.G(ctx).WithFields(log.Fields{
		"url": opts.url,
		"rev": opts.revision,
	}).Debug("cloning git repo")
	r, err := ggClone(ctx, memory.NewStorage(), opts.FS, cloneOpts)
	if err != nil {
		return nil, err
	}

	return &repo{Repository: r, auth: opts.Auth}, nil
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

// From strings like git@github.com:someOrg/someRepo.git or
// https://github.com/someOrg/someRepo?ref=someHash, extract
// the parts.
func parseGitUrl(n string) (host, orgRepo, path, ref, gitSuff string) {
	if strings.Contains(n, gitDelimiter) {
		index := strings.Index(n, gitDelimiter)
		// Adding _git/ to host
		host = normalizeGitHostSpec(n[:index+len(gitDelimiter)])
		orgRepo = strings.Split(strings.Split(n[index+len(gitDelimiter):], "/")[0], "?")[0]
		path, ref = peelQuery(n[index+len(gitDelimiter)+len(orgRepo):])
		return
	}

	host, n = parseHostSpec(n)
	gitSuff = gitSuffix
	if strings.Contains(n, gitSuffix) {
		index := strings.Index(n, gitSuffix)
		orgRepo = n[0:index]
		n = n[index+len(gitSuffix):]
		path, ref = peelQuery(n)
		return
	}

	i := strings.Index(n, "/")
	if i < 1 {
		path, ref = peelQuery(n)
		return
	}

	j := strings.Index(n[i+1:], "/")
	if j >= 0 {
		j += i + 1
		orgRepo = n[:j]
		path, ref = peelQuery(n[j+1:])
		return
	}

	path = ""
	orgRepo, ref = peelQuery(n)
	return
}

func peelQuery(arg string) (path, ref string) {
	parsed, err := url.Parse(arg)
	if err != nil {
		return path, ""
	}

	path = parsed.Path
	values := parsed.Query()
	branch := values.Get("ref")
	tag := values.Get("tag")
	sha := values.Get("sha")
	if sha != "" {
		ref = sha
		return
	}

	if tag != "" {
		ref = "refs/tags/" + tag
		return
	}

	if branch != "" {
		ref = "refs/heads/" + branch
		return
	}

	return
}

func parseHostSpec(n string) (string, string) {
	var host string
	// Start accumulating the host part.
	for _, p := range [...]string{
		// Order matters here.
		"git::", "gh:", "ssh://", "https://", "http://",
		"git@", "github.com:", "github.com/"} {
		if len(p) < len(n) && strings.ToLower(n[:len(p)]) == p {
			n = n[len(p):]
			host += p
		}
	}
	if host == "git@" {
		i := strings.Index(n, "/")
		if i > -1 {
			host += n[:i+1]
			n = n[i+1:]
		} else {
			i = strings.Index(n, ":")
			if i > -1 {
				host += n[:i+1]
				n = n[i+1:]
			}
		}
		return host, n
	}

	// If host is a http(s) or ssh URL, grab the domain part.
	for _, p := range [...]string{
		"ssh://", "https://", "http://"} {
		if strings.HasSuffix(host, p) {
			i := strings.Index(n, "/")
			if i > -1 {
				host = host + n[0:i+1]
				n = n[i+1:]
			}
			break
		}
	}

	return normalizeGitHostSpec(host), n
}

func normalizeGitHostSpec(host string) string {
	s := strings.ToLower(host)
	if strings.Contains(s, "github.com") {
		if strings.Contains(s, "git@") || strings.Contains(s, "ssh:") {
			host = "git@github.com:"
		} else {
			host = "https://github.com/"
		}
	}
	if strings.HasPrefix(s, "git::") {
		host = strings.TrimPrefix(s, "git::")
	}
	return host
}
