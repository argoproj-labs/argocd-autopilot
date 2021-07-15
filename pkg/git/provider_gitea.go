package git

import (
	"context"
	"fmt"

	gt "code.gitea.io/sdk/gitea"
)

//go:generate mockery --name Client --output gitea/mocks --case snake

type (
	Client interface {
		CreateOrgRepo(org string, opt gt.CreateRepoOption) (*gt.Repository, *gt.Response, error)
		CreateRepo(opt gt.CreateRepoOption) (*gt.Repository, *gt.Response, error)
		GetMyUserInfo() (*gt.User, *gt.Response, error)
	}

	gitea struct {
		client Client
	}
)

func newGitea(opts *ProviderOptions) (Provider, error) {
	c, err := gt.NewClient(opts.Host, gt.SetToken(opts.Auth.Password))
	if err != nil {
		return nil, err
	}

	g := &gitea{
		client: c,
	}

	return g, nil
}

func (g *gitea) CreateRepository(ctx context.Context, opts *CreateRepoOptions) (string, error) {
	authUser, res, err := g.client.GetMyUserInfo()
	if err != nil {
		if res.StatusCode == 401 {
			return "", ErrAuthenticationFailed(err)
		}

		return "", err
	}

	createOpts := gt.CreateRepoOption{
		Name:    opts.Name,
		Private: opts.Private,
	}

	var r *gt.Repository
	if authUser.UserName != opts.Owner {
		r, res, err = g.client.CreateOrgRepo(opts.Owner, createOpts)
	} else {
		r, res, err = g.client.CreateRepo(createOpts)
	}

	if err != nil {
		if res.StatusCode == 404 {
			return "", fmt.Errorf("owner %s not found: %w", opts.Owner, err)
		}

		return "", err
	}

	return r.CloneURL, nil
}
