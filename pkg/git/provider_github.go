package git

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	g "github.com/argoproj-labs/argocd-autopilot/pkg/git/github"

	gh "github.com/google/go-github/v43/github"
)

//go:generate mockgen -destination=./github/mocks/repos.go -package=mocks -source=./github/repos.go Repositories
//go:generate mockgen -destination=./github/mocks/users.go -package=mocks -source=./github/users.go Users

type github struct {
	opts         *ProviderOptions
	Repositories g.Repositories
	Users        g.Users
}

func newGithub(opts *ProviderOptions) (Provider, error) {
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

	if opts.Host != "" && !strings.Contains(opts.Host, "github.com") {
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

func (g *github) CreateRepository(ctx context.Context, orgRepo string) (string, error) {
	opts, err := getDefaultRepoOptions(orgRepo)
	if err != nil {
		return "", nil
	}

	authUser, err := g.getAuthenticatedUser(ctx)
	if err != nil {
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

func (g *github) GetAuthor(ctx context.Context) (username, email string, err error) {
	authUser, err := g.getAuthenticatedUser(ctx)
	if err != nil {
		return
	}

	username = authUser.GetName()
	if username == "" {
		username = authUser.GetLogin()
	}

	email = authUser.GetEmail()
	if email == "" {
		email = g.getEmail(ctx)
	}

	if email == "" {
		email = authUser.GetLogin()
	}

	return
}

func (g *github) getAuthenticatedUser(ctx context.Context) (*gh.User, error) {
	authUser, res, err := g.Users.Get(ctx, "")
	if err != nil {
		if res.StatusCode == 401 {
			return nil, ErrAuthenticationFailed(err)
		}

		return nil, err
	}

	return authUser, nil
}

func (g *github) getEmail(ctx context.Context) string {
	emails, _, err := g.Users.ListEmails(ctx, &gh.ListOptions{
		Page:    0,
		PerPage: 10,
	})
	if err != nil {
		return ""
	}

	var email *gh.UserEmail
	for _, e := range emails {
		if e.GetVisibility() != "public" {
			continue
		}

		if e.GetPrimary() && e.GetVerified() {
			email = e
			break
		}

		if e.GetPrimary() {
			email = e
		}

		if e.GetVerified() && !email.GetPrimary() {
			email = e
		}
	}

	return email.GetEmail()
}
