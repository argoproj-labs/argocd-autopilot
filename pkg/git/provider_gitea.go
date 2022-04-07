package git

import (
	"context"
	"fmt"

	gt "code.gitea.io/sdk/gitea"
)

//go:generate mockgen -destination=./gitea/mocks/client.go -package=mocks -source=./provider_gitea.go Client

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

func (g *gitea) CreateRepository(ctx context.Context, orgRepo string) (string, error) {
	opts, err := getDefaultRepoOptions(orgRepo)
	if err != nil {
		return "", nil
	}

	authUser, err := g.getAuthenticatedUser()
	if err != nil {
		return "", err
	}

	createOpts := gt.CreateRepoOption{
		Name:    opts.Name,
		Private: opts.Private,
	}

	var (
		r   *gt.Repository
		res *gt.Response
	)
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

func (g *gitea) GetAuthor(_ context.Context) (username, email string, err error) {
	authUser, err := g.getAuthenticatedUser()
	if err != nil {
		return
	}

	username = authUser.UserName
	email = authUser.Email
	return
}

func (g *gitea) getAuthenticatedUser() (*gt.User, error) {
	authUser, res, err := g.client.GetMyUserInfo()
	if err != nil {
		if res.StatusCode == 401 {
			return nil, ErrAuthenticationFailed(err)
		}

		return nil, err
	}

	return authUser, nil
}
