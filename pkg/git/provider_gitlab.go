package git

import (
	"context"
	"fmt"

	gl "github.com/xanzy/go-gitlab"
)

//go:generate mockery --name GitlabClient --output gitlab/mocks --case snake

type (
	GitlabClient interface {
		CurrentUser(options ...gl.RequestOptionFunc) (*gl.User, *gl.Response, error)
		CreateProject(opt *gl.CreateProjectOptions, options ...gl.RequestOptionFunc) (*gl.Project, *gl.Response, error)
		ListGroups(opt *gl.ListGroupsOptions, options ...gl.RequestOptionFunc) ([]*gl.Group, *gl.Response, error)
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
	c, err := gl.NewClient(opts.Auth.Password)
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

func (g *gitlab) CreateRepository(ctx context.Context, opts *CreateRepoOptions) (string, error) {
	authUser, res, err := g.client.CurrentUser()
	if err != nil {
		if res.StatusCode == 401 {
			return "", ErrAuthenticationFailed(err)
		}

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
		return "", fmt.Errorf("failed creating the project %s under %s: %w", opts.Name, opts.Owner, err)
	}

	if p.WebURL == "" {
		return "", fmt.Errorf("project url is empty")
	}

	return p.WebURL, err
}

func (g *gitlab) getGroupIdByName(groupName string) (int, error) {
	groups, _, err := g.client.ListGroups(&gl.ListGroupsOptions{
		MinAccessLevel: gl.AccessLevel(gl.DeveloperPermissions),
	})
	if err != nil {
		return 0, err
	}

	for _, group := range groups {
		if group.Path == groupName {
			return group.ID, nil
		}
	}
	return 0, fmt.Errorf("group %s not found", groupName)
}
