package git

import (
	"context"
	"fmt"
	"io/ioutil"

	"net/http"

	cferrors "github.com/codefresh-io/cf-argo/pkg/errors"
	"github.com/codefresh-io/cf-argo/pkg/log"

	gh "github.com/google/go-github/v32/github"
)

type github struct {
	opts   *Options
	client *gh.Client
}

func newGithub(opts *Options) (Provider, error) {
	var c *gh.Client
	var err error
	hc := &http.Client{}

	if opts.Auth != nil {
		hc.Transport = &gh.BasicAuthTransport{
			Username: opts.Auth.Username,
			Password: opts.Auth.Password,
		}
	}

	if opts.Host != "" {
		c, err = gh.NewEnterpriseClient(opts.Host, opts.Host, hc)
		if err != nil {
			return nil, err
		}
	} else {
		c = gh.NewClient(hc)
	}

	g := &github{
		opts:   opts,
		client: c,
	}
	return g, nil
}

func (g *github) CreateRepository(ctx context.Context, opts *CreateRepositoryOptions) (string, error) {
	l := log.G(ctx).WithFields(log.Fields{
		"owner": opts.Owner,
		"repo":  opts.Name,
	})

	l.Debug("creating repository")

	authUser, _, err := g.client.Users.Get(ctx, "") // get authenticated user details
	if err != nil {
		return "", err
	}

	org := ""
	if *authUser.Login != opts.Owner {
		org = opts.Owner
	}

	r, _, err := g.client.Repositories.Create(ctx, org, &gh.Repository{
		Name:    gh.String(opts.Name),
		Private: gh.Bool(opts.Private),
	})
	if err != nil {
		return "", err
	}

	if r.CloneURL == nil {
		return "", fmt.Errorf("repo clone url is nil")
	}

	l.Debug("repository created")

	return *r.CloneURL, err
}

func (g *github) CloneRepository(ctx context.Context, opts *GetRepositoryOptions) (Repository, error) {
	cloneURL, err := g.getRepository(ctx, opts)
	if err != nil {
		return nil, err
	}

	log.G(ctx).Debug("creating temp dir for gitops repo")
	clonePath, err := ioutil.TempDir("", "repo-")
	cferrors.CheckErr(err)
	log.G(ctx).WithField("location", clonePath).Debug("temp dir created")

	log.G(ctx).Printf("cloning existing gitops repository...")

	return g.clone(ctx, &CloneOptions{
		URL:  cloneURL,
		Path: clonePath,
	})
}

func (g *github) clone(ctx context.Context, opts *CloneOptions) (Repository, error) {
	if opts == nil {
		return nil, cferrors.ErrNilOpts
	}

	auth := g.opts.Auth
	if opts.Auth != nil {
		auth = opts.Auth
	}
	return Clone(ctx, &CloneOptions{
		URL:  opts.URL,
		Path: opts.Path,
		Auth: auth,
	})
}

func (g *github) getRepository(ctx context.Context, opts *GetRepositoryOptions) (string, error) {
	if opts == nil {
		return "", cferrors.ErrNilOpts
	}

	r, res, err := g.client.Repositories.Get(ctx, opts.Owner, opts.Name)
	if err != nil && res == nil {
		return "", err
	}

	if res.StatusCode == 404 {
		return "", ErrRepoNotFound
	}

	return *r.CloneURL, nil
}
