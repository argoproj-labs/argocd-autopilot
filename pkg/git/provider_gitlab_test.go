package git

import (
	"context"
	"errors"
	"net/http"
	"testing"

	glmocks "github.com/argoproj-labs/argocd-autopilot/pkg/git/gitlab/mocks"
	gl "github.com/xanzy/go-gitlab"

	"github.com/stretchr/testify/assert"
)

func Test_gitlab_CreateRepository(t *testing.T) {
	tests := map[string]struct {
		opts     *CreateRepoOptions
		beforeFn func(*glmocks.GitlabClient)
		want     string
		wantErr  string
	}{
		"Fails if credentials are wrong": {
			opts: &CreateRepoOptions{
				Name:  "projectName",
				Owner: "username",
			},
			beforeFn: func(c *glmocks.GitlabClient) {
				res := &gl.Response{
					Response: &http.Response{
						StatusCode: 401,
					},
				}
				c.On("CurrentUser").Return(nil, res, errors.New("some error"))
			},
			wantErr: "authentication failed, make sure credentials are correct: some error",
		},
		"Fails if can't find current user": {
			opts: &CreateRepoOptions{
				Name:  "projectName",
				Owner: "username",
			},
			beforeFn: func(c *glmocks.GitlabClient) {
				res := &gl.Response{
					Response: &http.Response{},
				}
				c.On("CurrentUser").Return(nil, res, errors.New("some error"))
			},
			wantErr: "some error",
		},
		"Fails if can't find group": {
			opts: &CreateRepoOptions{
				Name:  "projectName",
				Owner: "org",
			},
			beforeFn: func(c *glmocks.GitlabClient) {
				u := &gl.User{Username: "username"}
				c.On("CurrentUser").Return(u, nil, nil)
				g := []*gl.Group{&gl.Group{Path: "anotherOrg", ID: 1}}
				c.On("ListGroups", &gl.ListGroupsOptions{
					MinAccessLevel: gl.AccessLevel(gl.DeveloperPermissions),
				}).Return(g, nil, nil)
			},
			wantErr: "group org not found",
		},
		"Fails if can't create project": {
			opts: &CreateRepoOptions{
				Name:  "projectName",
				Owner: "username",
			},
			beforeFn: func(c *glmocks.GitlabClient) {
				u := &gl.User{Username: "username"}
				c.On("CurrentUser").Return(u, nil, nil)
				createOpts := gl.CreateProjectOptions{
					Name:       gl.String("projectName"),
					Visibility: gl.Visibility(gl.PublicVisibility),
				}
				res := &gl.Response{
					Response: &http.Response{},
				}
				c.On("CreateProject", &createOpts).Return(nil, res, errors.New("some error"))
			},
			wantErr: "failed creating the project projectName under username: some error",
		},
		"Creates project under user": {
			opts: &CreateRepoOptions{
				Name:  "projectName",
				Owner: "username",
			},
			beforeFn: func(c *glmocks.GitlabClient) {
				u := &gl.User{Username: "username"}
				c.On("CurrentUser").Return(u, nil, nil)
				p := &gl.Project{WebURL: "http://gitlab.com/username/projectName"}
				createOpts := gl.CreateProjectOptions{
					Name:       gl.String("projectName"),
					Visibility: gl.Visibility(gl.PublicVisibility),
				}
				c.On("CreateProject", &createOpts).Return(p, nil, nil)
			},
			want: "http://gitlab.com/username/projectName",
		},
		"Creates project under group": {
			opts: &CreateRepoOptions{
				Name:  "projectName",
				Owner: "org",
			},
			beforeFn: func(c *glmocks.GitlabClient) {
				u := &gl.User{Username: "username"}
				c.On("CurrentUser").Return(u, nil, nil)
				p := &gl.Project{WebURL: "http://gitlab.com/org/projectName"}
				g := []*gl.Group{&gl.Group{Path: "org", ID: 1}}
				c.On("ListGroups", &gl.ListGroupsOptions{
					MinAccessLevel: gl.AccessLevel(gl.DeveloperPermissions),
				}).Return(g, nil, nil)
				createOpts := gl.CreateProjectOptions{
					Name:        gl.String("projectName"),
					Visibility:  gl.Visibility(gl.PublicVisibility),
					NamespaceID: gl.Int(1),
				}
				c.On("CreateProject", &createOpts).Return(p, nil, nil)
			},
			want: "http://gitlab.com/org/projectName",
		},
		"Creates private project": {
			opts: &CreateRepoOptions{
				Name:    "projectName",
				Owner:   "username",
				Private: true,
			},
			beforeFn: func(c *glmocks.GitlabClient) {
				u := &gl.User{Username: "username"}
				c.On("CurrentUser").Return(u, nil, nil)
				p := &gl.Project{WebURL: "http://gitlab.com/username/projectName"}
				createOpts := gl.CreateProjectOptions{
					Name:       gl.String("projectName"),
					Visibility: gl.Visibility(gl.PrivateVisibility),
				}
				res := &gl.Response{
					Response: &http.Response{},
				}
				c.On("CreateProject", &createOpts).Return(p, res, nil)
			},
			want: "http://gitlab.com/username/projectName",
		},
		"Fails when no WebURL": {
			opts: &CreateRepoOptions{
				Name:  "projectName",
				Owner: "username",
			},
			beforeFn: func(c *glmocks.GitlabClient) {
				u := &gl.User{Username: "username"}
				c.On("CurrentUser").Return(u, nil, nil)
				p := &gl.Project{WebURL: ""}
				createOpts := gl.CreateProjectOptions{
					Name:       gl.String("projectName"),
					Visibility: gl.Visibility(gl.PublicVisibility),
				}
				res := &gl.Response{
					Response: &http.Response{},
				}
				c.On("CreateProject", &createOpts).Return(p, res, nil)
			},
			wantErr: "project url is empty",
			want:    "",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := &glmocks.GitlabClient{}
			tt.beforeFn(mockClient)
			g := &gitlab{
				client: mockClient,
			}
			got, err := g.CreateRepository(context.Background(), tt.opts)

			mockClient.AssertExpectations(t)
			if err != nil {
				if tt.wantErr != "" {
					assert.EqualError(t, err, tt.wantErr)
				} else {
					t.Errorf("gitlab.CreateRepository() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			if got != tt.want {
				t.Errorf("gitlab.CreateRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}
