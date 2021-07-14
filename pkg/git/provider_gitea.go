package git

import (
	"context"
	"fmt"

	gt "code.gitea.io/sdk/gitea"
)

type gitea struct {
	opts   *ProviderOptions
	client *gt.Client
}

func newGitea(opts *ProviderOptions) (Provider, error) {
	c, err := gt.NewClient(opts.Host, gt.SetToken(opts.Auth.Password))
	if err != nil {
		return nil, err
	}

	g := &gitea{
		opts:   opts,
		client: c,
	}

	return g, nil
}

func (g *gitea) CreateRepository(ctx context.Context, opts *CreateRepoOptions) (string, error) {
	r, res, err := g.client.CreateRepo(gt.CreateRepoOption{
		Name:    opts.Name,
		Private: opts.Private,
	})
	if err != nil {
		if res.StatusCode == 404 {
			return "", fmt.Errorf("owner %s not found: %w", opts.Owner, err)
		}

		return "", err
	}

	return r.CloneURL, nil
}
