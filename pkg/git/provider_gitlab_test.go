package git

import (
	"context"
	"errors"
	"net/http"
	"testing"

	glmocks "github.com/argoproj-labs/argocd-autopilot/pkg/git/gitlab/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	gl "github.com/xanzy/go-gitlab"
)

func Test_gitlab_CreateRepository(t *testing.T) {
	tests := map[string]struct {
		orgRepo  string
		beforeFn func(*glmocks.MockGitlabClient)
		want     string
		wantErr  string
	}{
		"Fails if credentials are wrong": {
			orgRepo: "username/projectName",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				res := &gl.Response{
					Response: &http.Response{
						StatusCode: 401,
					},
				}
				c.EXPECT().CurrentUser().
					Times(1).
					Return(nil, res, errors.New("some error"))
			},
			wantErr: "authentication failed, make sure credentials are correct: some error",
		},
		"Fails if can't find current user": {
			orgRepo: "username/projectName",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				res := &gl.Response{
					Response: &http.Response{},
				}
				c.EXPECT().CurrentUser().
					Times(1).
					Return(nil, res, errors.New("some error"))
			},
			wantErr: "some error",
		},
		"Fails if can't find group": {
			orgRepo: "org/projectName",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				u := &gl.User{Username: "username"}
				g := []*gl.Group{{FullPath: "anotherOrg", ID: 1}}

				c.EXPECT().CurrentUser().
					Times(1).
					Return(u, nil, nil)
				c.EXPECT().ListGroups(&gl.ListGroupsOptions{
					MinAccessLevel: gl.AccessLevel(gl.DeveloperPermissions),
					TopLevelOnly:   gl.Bool(false),
				}).
					Times(1).
					Return(g, nil, nil)
			},
			wantErr: "group org not found",
		},
		"Fails if can't create project": {
			orgRepo: "username/projectName",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				u := &gl.User{Username: "username"}
				createOpts := gl.CreateProjectOptions{
					Name:       gl.String("projectName"),
					Visibility: gl.Visibility(gl.PrivateVisibility),
				}
				res := &gl.Response{
					Response: &http.Response{},
				}

				c.EXPECT().CurrentUser().
					Times(1).
					Return(u, nil, nil)

				c.EXPECT().CreateProject(&createOpts).
					Times(1).
					Return(nil, res, errors.New("some error"))
			},
			wantErr: "failed creating the project projectName under username: some error",
		},
		"Creates project under user": {
			orgRepo: "username/projectName",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				u := &gl.User{Username: "username"}
				p := &gl.Project{WebURL: "http://gitlab.com/username/projectName"}
				createOpts := gl.CreateProjectOptions{
					Name:       gl.String("projectName"),
					Visibility: gl.Visibility(gl.PrivateVisibility),
				}

				c.EXPECT().CurrentUser().
					Times(1).
					Return(u, nil, nil)
				c.EXPECT().CreateProject(&createOpts).
					Times(1).
					Return(p, nil, nil)
			},
			want: "http://gitlab.com/username/projectName",
		},
		"Creates project under group": {
			orgRepo: "org/projectName",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				u := &gl.User{Username: "username"}
				c.EXPECT().CurrentUser().Return(u, nil, nil)
				p := &gl.Project{WebURL: "http://gitlab.com/org/projectName"}
				g := []*gl.Group{{FullPath: "org", ID: 1}}
				createOpts := gl.CreateProjectOptions{
					Name:        gl.String("projectName"),
					Visibility:  gl.Visibility(gl.PrivateVisibility),
					NamespaceID: gl.Int(1),
				}

				c.EXPECT().ListGroups(&gl.ListGroupsOptions{
					MinAccessLevel: gl.AccessLevel(gl.DeveloperPermissions),
					TopLevelOnly:   gl.Bool(false),
				}).
					Times(1).
					Return(g, nil, nil)

				c.EXPECT().CreateProject(&createOpts).
					Times(1).
					Return(p, nil, nil)
			},
			want: "http://gitlab.com/org/projectName",
		},
		"Creates project under sub group": {
			orgRepo: "org/subOrg/projectName",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				u := &gl.User{Username: "username"}
				c.EXPECT().CurrentUser().Return(u, nil, nil)
				p := &gl.Project{WebURL: "http://gitlab.com/org/subOrg/projectName"}
				g := []*gl.Group{{FullPath: "org/subOrg", ID: 1}}
				createOpts := gl.CreateProjectOptions{
					Name:        gl.String("projectName"),
					Visibility:  gl.Visibility(gl.PrivateVisibility),
					NamespaceID: gl.Int(1),
				}

				c.EXPECT().ListGroups(&gl.ListGroupsOptions{
					MinAccessLevel: gl.AccessLevel(gl.DeveloperPermissions),
					TopLevelOnly:   gl.Bool(false),
				}).
					Times(1).
					Return(g, nil, nil)

				c.EXPECT().CreateProject(&createOpts).
					Times(1).
					Return(p, nil, nil)
			},
			want: "http://gitlab.com/org/subOrg/projectName",
		},
		"Creates private project": {
			orgRepo: "username/projectName",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				u := &gl.User{Username: "username"}
				p := &gl.Project{WebURL: "http://gitlab.com/username/projectName"}
				createOpts := gl.CreateProjectOptions{
					Name:       gl.String("projectName"),
					Visibility: gl.Visibility(gl.PrivateVisibility),
				}
				res := &gl.Response{
					Response: &http.Response{},
				}

				c.EXPECT().CurrentUser().
					Times(1).
					Return(u, nil, nil)
				c.EXPECT().CreateProject(&createOpts).
					Times(1).
					Return(p, res, nil)
			},
			want: "http://gitlab.com/username/projectName",
		},
		"Fails when no WebURL": {
			orgRepo: "username/projectName",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				u := &gl.User{Username: "username"}
				p := &gl.Project{WebURL: ""}
				createOpts := gl.CreateProjectOptions{
					Name:       gl.String("projectName"),
					Visibility: gl.Visibility(gl.PrivateVisibility),
				}
				res := &gl.Response{
					Response: &http.Response{},
				}
				c.EXPECT().CurrentUser().
					Times(1).
					Return(u, nil, nil)
				c.EXPECT().CreateProject(&createOpts).
					Times(1).
					Return(p, res, nil)
			},
			wantErr: "project url is empty",
			want:    "",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := glmocks.NewMockGitlabClient(gomock.NewController(t))
			tt.beforeFn(mockClient)
			g := &gitlab{
				client: mockClient,
			}
			got, err := g.CreateRepository(context.Background(), tt.orgRepo)

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
