package git

import (
	"context"
	"fmt"
	"net/http"

	"github.com/argoproj-labs/argocd-autopilot/pkg/util"
	gl "github.com/xanzy/go-gitlab"
)

//go:generate mockgen -destination=./gitlab/mocks/client.go -package=mocks -source=./provider_gitlab.go GitlabClient

type (
	GitlabClient interface {
		CurrentUser(options ...gl.RequestOptionFunc) (*gl.User, *gl.Response, error)
		CreateProject(opt *gl.CreateProjectOptions, options ...gl.RequestOptionFunc) (*gl.Project, *gl.Response, error)
		GetProject(pid interface{}, opt *gl.GetProjectOptions, options ...gl.RequestOptionFunc) (*gl.Project, *gl.Response, error)
		GetGroup(gid interface{}, opt *gl.GetGroupOptions, options ...gl.RequestOptionFunc) (*gl.Group, *gl.Response, error)
	}

	clientImpl struct {
		gl.ProjectsService
		gl.UsersService
		gl.GroupsService
	}

	gitlab struct {
		opts   *ProviderOptions
		client GitlabClient
	}
)

func newGitlab(opts *ProviderOptions) (Provider, error) {
	host, _, _, _, _, _, _ := util.ParseGitUrl(opts.RepoURL)
	transport, err := DefaultTransportWithCa(opts.Auth.CertFile)
	if err != nil {
		return nil, err
	}

	c, err := gl.NewClient(
		opts.Auth.Password,
		gl.WithBaseURL(host),
		gl.WithHTTPClient(&http.Client{
			Transport: transport,
		}),
	)
	if err != nil {
		return nil, err
	}

	g := &gitlab{
		opts: opts,
		client: &clientImpl{
			ProjectsService: *c.Projects,
			UsersService:    *c.Users,
			GroupsService:   *c.Groups,
		},
	}

	return g, nil
}

func (g *gitlab) CreateRepository(ctx context.Context, orgRepo string) (defaultBranch string, err error) {
	opts, err := getDefaultRepoOptions(orgRepo)
	if err != nil {
		return "", err
	}

	authUser, err := g.getAuthenticatedUser()
	if err != nil {
		return "", err
	}

	createOpts := gl.CreateProjectOptions{
		Name:       &opts.Name,
		Visibility: gl.Visibility(gl.PublicVisibility),
	}

	if opts.Private {
		createOpts.Visibility = gl.Visibility(gl.PrivateVisibility)
	}

	if authUser.Username != opts.Owner {
		groupId, err := g.getGroupIdByName(opts.Owner)
		if err != nil {
			return "", err
		}

		createOpts.NamespaceID = gl.Int(groupId)
	}

	p, _, err := g.client.CreateProject(&createOpts)
	if err != nil {
		return "", fmt.Errorf("failed creating the project \"%s\" under \"%s\": %w", opts.Name, opts.Owner, err)
	}

	return p.DefaultBranch, err
}

func (g *gitlab) GetDefaultBranch(ctx context.Context, orgRepo string) (string, error) {
	opts, err := getDefaultRepoOptions(orgRepo)
	if err != nil {
		return "", err
	}

	p, res, err := g.client.GetProject(orgRepo, &gl.GetProjectOptions{})
	if err != nil {
		if res != nil && res.StatusCode == 404 {
			return "", fmt.Errorf("owner \"%s\" not found: %w", opts.Owner, err)
		}

		return "", err
	}

	return p.DefaultBranch, nil
}

func (g *gitlab) GetAuthor(_ context.Context) (username, email string, err error) {
	authUser, err := g.getAuthenticatedUser()
	if err != nil {
		return
	}

	username = authUser.Name
	if username == "" {
		username = authUser.Username
	}

	email = authUser.Email
	if email == "" {
		email = authUser.Username
	}

	return
}

func (g *gitlab) getAuthenticatedUser() (*gl.User, error) {
	authUser, res, err := g.client.CurrentUser()
	if err != nil {
		if res != nil && res.StatusCode == 401 {
			return nil, ErrAuthenticationFailed(err)
		}

		return nil, err
	}

	return authUser, nil
}

func (g *gitlab) getGroupIdByName(groupName string) (int, error) {
	group, _, err := g.client.GetGroup(groupName, &gl.GetGroupOptions{})
	if err != nil {
		return 0, err
	}

	return group.ID, nil
}
