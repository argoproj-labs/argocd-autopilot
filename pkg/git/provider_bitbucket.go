package git

import (
	"context"
	"errors"
	"fmt"

	bb "github.com/ktrysmt/go-bitbucket"
)

//go:generate mockgen -destination=./bitbucket/mocks/client.go -package=mocks -source=./provider_bitbucket.go bbRepo bbUser

type (
	bitbucket struct {
		opts       *ProviderOptions
		Repository bbRepo
		User       bbUser
	}

	bbRepo interface {
		Create(ro *bb.RepositoryOptions) (*bb.Repository, error)
		Get(ro *bb.RepositoryOptions) (*bb.Repository, error)
	}

	bbUser interface {
		Profile() (*bb.User, error)
		Emails() (interface{}, error)
	}
)

func newBitbucket(opts *ProviderOptions) (Provider, error) {
	var err error
	c := bb.NewBasicAuth(opts.Auth.Username, opts.Auth.Password)
	if c == nil {
		return nil, errors.New("Authentication info is invalid")
	}

	c.HttpClient.Transport, err = DefaultTransportWithCa(opts.Auth.CertFile)
	if err != nil {
		return nil, err
	}

	g := &bitbucket{
		opts:       opts,
		Repository: c.Repositories.Repository,
		User:       c.User,
	}

	return g, nil
}

func (g *bitbucket) CreateRepository(ctx context.Context, orgRepo string) (defaultBranch string, err error) {
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

	p, err := g.Repository.Create(createOpts)

	if err != nil {
		return "", fmt.Errorf("failed creating the repository \"%s\" under \"%s\": %w", opts.Name, opts.Owner, err)
	}

	return p.Mainbranch.Name, err
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

	repo, err := g.Repository.Get(repoOpts)
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
	user, err := g.User.Profile()

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (g *bitbucket) getAuthenticatedUserEmail() (string, error) {
	emails, err := g.User.Emails()
	if err != nil {
		return "", err
	}

	userEmails := emails.(map[string]interface{})
	var lastEmailInfo map[string]interface{}

	for _, emailValues := range userEmails["values"].([]interface{}) {
		emailInfo := emailValues.(map[string]interface{})
		isPrimary := emailInfo["is_primary"].(bool)
		isConfirmed := emailInfo["is_confirmed"].(bool)
		isLastPrimary, lastExist := lastEmailInfo["is_primary"].(bool)
		if isConfirmed && isPrimary {
			lastEmailInfo = emailInfo
			break
		}

		if isPrimary {
			lastEmailInfo = emailInfo
		}

		if ((lastExist && !isLastPrimary) || !lastExist) && isConfirmed {
			lastEmailInfo = emailInfo
		}
	}

	if email, ok := lastEmailInfo["email"].(string); ok {
		return email, nil
	}

	return "", nil
}
