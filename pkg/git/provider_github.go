package git

import (
	"context"
	"fmt"
	"net/http"

	g "github.com/argoproj/argocd-autopilot/pkg/git/github"
	gh "github.com/google/go-github/v34/github"
)

//go:generate interfacer -for github.com/google/go-github/v34/github.RepositoriesService -as github.Repositories -o github/repos.go
//go:generate interfacer -for github.com/google/go-github/v34/github.UsersService -as github.Users -o github/users.go
//go:generate mockery -dir github -all
type github struct {
	opts         *Options
	Repositories g.Repositories
	Users        g.Users
}

func newGithub(opts *Options) (Provider, error) {
	var (
		c   *gh.Client
		err error
	)

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
		opts:         opts,
		Repositories: c.Repositories,
		Users:        c.Users,
	}

	return g, nil
}

func (g *github) GetRepository(ctx context.Context, opts *GetRepoOptions) (string, error) {
	r, res, err := g.Repositories.Get(ctx, opts.Owner, opts.Name)
	if res != nil && res.StatusCode == http.StatusNotFound {
		return "", ErrRepoNotFound
	}

	if err != nil {
		return "", err
	}

	return *r.CloneURL, nil
}

func (g *github) CreateRepository(ctx context.Context, opts *CreateRepoOptions) (string, error) {
	authUser, res, err := g.Users.Get(ctx, "") // get authenticated user details
	if err != nil {
		if res.StatusCode == 401 {
			return "", ErrAuthenticationFailed(err)
		}
		return "", err
	}

	org := ""
	if *authUser.Login != opts.Owner {
		org = opts.Owner
	}

	r, res, err := g.Repositories.Create(ctx, org, &gh.Repository{
		Name:    gh.String(opts.Name),
		Private: gh.Bool(opts.Private),
	})
	if err != nil {
		if res.StatusCode == 404 {
			return "", fmt.Errorf("owner %s not found: %w", opts.Owner, err)
		}
		return "", err
	}

	if r.CloneURL == nil {
		return "", fmt.Errorf("repo clone url is nil")
	}

	return *r.CloneURL, err
}
