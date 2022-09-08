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
			wantErr: "authentication failed, make sure credentials are correct: some error",
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
		},
		"Fails if can't find current user": {
			orgRepo: "username/projectName",
			wantErr: "some error",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				res := &gl.Response{
					Response: &http.Response{},
				}
				c.EXPECT().CurrentUser().
					Times(1).
					Return(nil, res, errors.New("some error"))
			},
		},
		"Fails if can't find group": {
			orgRepo: "org/projectName",
			wantErr: "some error",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				u := &gl.User{Username: "username"}

				c.EXPECT().CurrentUser().
					Times(1).
					Return(u, nil, nil)
				c.EXPECT().GetGroup("org", &gl.GetGroupOptions{}).
					Times(1).
					Return(nil, nil, errors.New("some error"))
			},
		},
		"Fails if can't create project": {
			orgRepo: "username/projectName",
			wantErr: "failed creating the project \"projectName\" under \"username\": some error",
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
		},
		"Creates project under user": {
			orgRepo: "username/projectName",
			want:    "main",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				u := &gl.User{Username: "username"}
				p := &gl.Project{
					DefaultBranch: "main",
				}
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
		},
		"Creates project under group": {
			orgRepo: "org/projectName",
			want:    "main",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				u := &gl.User{Username: "username"}
				c.EXPECT().CurrentUser().Return(u, nil, nil)
				p := &gl.Project{
					DefaultBranch: "main",
				}
				g := &gl.Group{FullPath: "org", ID: 1}
				createOpts := gl.CreateProjectOptions{
					Name:        gl.String("projectName"),
					Visibility:  gl.Visibility(gl.PrivateVisibility),
					NamespaceID: gl.Int(1),
				}

				c.EXPECT().GetGroup("org", &gl.GetGroupOptions{}).
					Times(1).
					Return(g, nil, nil)

				c.EXPECT().CreateProject(&createOpts).
					Times(1).
					Return(p, nil, nil)
			},
		},
		"Creates project under sub group": {
			orgRepo: "org/subOrg/projectName",
			want:    "main",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				u := &gl.User{Username: "username"}
				c.EXPECT().CurrentUser().Return(u, nil, nil)
				p := &gl.Project{
					DefaultBranch: "main",
				}
				g := &gl.Group{FullPath: "org/subOrg", ID: 1}
				createOpts := gl.CreateProjectOptions{
					Name:        gl.String("projectName"),
					Visibility:  gl.Visibility(gl.PrivateVisibility),
					NamespaceID: gl.Int(1),
				}

				c.EXPECT().GetGroup("org/subOrg", &gl.GetGroupOptions{}).
					Times(1).
					Return(g, nil, nil)

				c.EXPECT().CreateProject(&createOpts).
					Times(1).
					Return(p, nil, nil)
			},
		},
		"Creates private project": {
			orgRepo: "username/projectName",
			want:    "main",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				u := &gl.User{Username: "username"}
				p := &gl.Project{
					DefaultBranch: "main",
				}
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
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equalf(t, tt.want, got, "CreateRepository - %s", name)
		})
	}
}

func Test_gitlab_GetDefaultBranch(t *testing.T) {
	tests := map[string]struct {
		orgRepo  string
		want     string
		wantErr  string
		beforeFn func(*glmocks.MockGitlabClient)
	}{
		"Should fail if orgRepo is invalid": {
			orgRepo: "invalid",
			wantErr: "failed parsing organization and repo from 'invalid'",
		},
		"Should fail if repo Get fails with 401": {
			orgRepo: "owner/repo",
			wantErr: "some error",
			beforeFn: func(mc *glmocks.MockGitlabClient) {
				res := &gl.Response{
					Response: &http.Response{
						StatusCode: 401,
					},
				}
				mc.EXPECT().GetProject("owner/repo", &gl.GetProjectOptions{}).Times(1).Return(nil, res, errors.New("some error"))
			},
		},
		"Should fail if repo Get fails with 404": {
			orgRepo: "owner/repo",
			wantErr: "owner \"owner\" not found: some error",
			beforeFn: func(mc *glmocks.MockGitlabClient) {
				res := &gl.Response{
					Response: &http.Response{
						StatusCode: 404,
					},
				}
				mc.EXPECT().GetProject("owner/repo", &gl.GetProjectOptions{}).Times(1).Return(nil, res, errors.New("some error"))
			},
		},
		"Should succeed with valid default branch": {
			orgRepo: "owner/repo",
			want:    "main",
			beforeFn: func(mc *glmocks.MockGitlabClient) {
				r := &gl.Project{
					DefaultBranch: "main",
				}
				mc.EXPECT().GetProject("owner/repo", &gl.GetProjectOptions{}).Times(1).Return(r, nil, nil)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := glmocks.NewMockGitlabClient(gomock.NewController(t))
			if tt.beforeFn != nil {
				tt.beforeFn(mockClient)
			}

			g := &gitlab{
				client: mockClient,
			}
			got, err := g.GetDefaultBranch(context.Background(), tt.orgRepo)
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_gitlab_GetAuthor(t *testing.T) {
	tests := map[string]struct {
		wantUsername string
		wantEmail    string
		wantErr      string
		beforeFn     func(*glmocks.MockGitlabClient)
	}{
		"Should fail with auth failed if user GET returns 401": {
			wantErr: "authentication failed, make sure credentials are correct: some error",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				c.EXPECT().CurrentUser().Times(1).
					Return(nil, &gl.Response{
						Response: &http.Response{
							StatusCode: 401,
						},
					}, errors.New("some error"))
			},
		},
		"Should fail if user GET returns 404": {
			wantErr: "some error",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				c.EXPECT().CurrentUser().Times(1).
					Return(nil, &gl.Response{
						Response: &http.Response{
							StatusCode: 404,
						},
					}, errors.New("some error"))
			},
		},
		"Should return name and email if available": {
			wantUsername: "name",
			wantEmail:    "name@email",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				c.EXPECT().CurrentUser().Times(1).Return(&gl.User{
					Name:  "name",
					Email: "name@email",
				}, nil, nil)
			},
		},
		"Should return username no displayName and emailAddress": {
			wantUsername: "username",
			wantEmail:    "username",
			beforeFn: func(c *glmocks.MockGitlabClient) {
				c.EXPECT().CurrentUser().Times(1).Return(&gl.User{
					Username: "username",
				}, nil, nil)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockClient := glmocks.NewMockGitlabClient(gomock.NewController(t))
			if tt.beforeFn != nil {
				tt.beforeFn(mockClient)
			}

			g := &gitlab{
				client: mockClient,
			}
			gotUsername, gotEmail, err := g.GetAuthor(context.Background())
			if err != nil || tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.wantUsername, gotUsername, "username mismatch")
			assert.Equal(t, tt.wantEmail, gotEmail, "email mismatch")
		})
	}
}
