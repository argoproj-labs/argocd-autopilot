package git

import (
	"context"
	"errors"
	"fmt"

	bb "github.com/ktrysmt/go-bitbucket"
)

//go:generate mockgen -destination=./gitlab/mocks/client.go -package=mocks -source=./provider_gitlab.go GitlabClient

type bitbucket struct {
	opts   *ProviderOptions
	client *bb.Client
}

func newBitbucket(opts *ProviderOptions) (Provider, error) {
	c := bb.NewBasicAuth(opts.Auth.Username, opts.Auth.Password)

	if c == nil {
		return nil, errors.New("Authentication info is invalid")
	}
	g := &bitbucket{
		opts:   opts,
		client: c,
	}

	return g, nil

}

func (g *bitbucket) CreateRepository(ctx context.Context, orgRepo string) (string, error) {
	opts, err := getDefaultRepoOptions(orgRepo)
	if err != nil {
		return "", err
	}

	createOpts := &bb.RepositoryOptions{
		Owner:    opts.Owner,
		RepoSlug: opts.Name,
		Scm:      "git",
	}

	if opts.Private {
		createOpts.IsPrivate = fmt.Sprintf("%t", opts.Private)
	}

	p, err := g.client.Repositories.Repository.Create(createOpts)
	if err != nil {
		return "", fmt.Errorf("failed creating the repository \"%s\" under \"%s\": %w", opts.Name, opts.Owner, err)
	}

	var cloneUrl string
	cloneLinksObj := p.Links["clone"]
	for _, cloneLink := range cloneLinksObj.([]interface{}) {
		if link, ok := cloneLink.(map[string]interface{}); ok {
			if link["name"].(string) == "https" {
				cloneUrl = link["href"].(string)
			}
		}
	}

	if cloneUrl == "" {
		return "", fmt.Errorf("clone url is empty")
	}

	return cloneUrl, err
}

func (g *bitbucket) GetDefaultBranch(ctx context.Context, orgRepo string) (string, error) {
	opts, err := getDefaultRepoOptions(orgRepo)
	if err != nil {
		return "", err
	}

	repoOpts := &bb.RepositoryOptions{
		Owner:    opts.Owner,
		RepoSlug: opts.Name,
		Scm:      "git",
	}

	if opts.Private {
		repoOpts.IsPrivate = fmt.Sprintf("%t", opts.Private)
	}

	repo, err := g.client.Repositories.Repository.Get(repoOpts)

	if err != nil {
		return "", err
	}

	return repo.Mainbranch.Name, nil

}

func (g *bitbucket) GetAuthor(_ context.Context) (username, email string, err error) {
	authUser, err := g.getAuthenticatedUser()
	if err != nil {
		return
	}

	username = authUser.Username

	authUserEmail, err := g.getAuthenticatedUserEmail()
	if err != nil || authUserEmail == "" {
		email = authUser.Username
		return
	}

	email = authUserEmail

	return
}

func (g *bitbucket) getAuthenticatedUser() (*bb.User, error) {
	user, err := g.client.User.Profile()

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (g *bitbucket) getAuthenticatedUserEmail() (string, error) {
	emails, err := g.client.User.Emails()

	if err != nil {
		return "", err
	}

	userEmails, ok := emails.(map[string]interface{})
	if !ok {
		return "", nil
	}

	for _, emailValues := range userEmails["values"].([]interface{}) {
		if emailStrValues, ok := emailValues.(map[string]interface{}); ok {
			if email, ok := emailStrValues["email"].(string); ok {
				return email, nil
			}
		}
	}

	return "", nil
}
